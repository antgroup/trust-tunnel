# Trust-Tunnel 代码质量重构设计

> 创建日期: 2026-03-04

## 背景

Trust-Tunnel 是一个安全隧道工具，用于创建到远程容器和物理主机的安全连接。当前代码功能完整，但代码水位需要全面提升，包括可读性、可维护性、健壮性和可测试性。

## 目标

全面提升代码质量，使其更易于理解、测试、维护和扩展。

## 约束条件

| 约束 | 说明 |
|------|------|
| 对外接口兼容 | CLI 参数、配置格式、API 不能变更 |
| 渐进式交付 | 分多个小 PR，每个可独立合并 |
| 功能等价 | 重构后功能必须与原有一致 |

## 重构策略

采用**分层重构**方式，按质量维度分阶段推进：

```
阶段一：可读性 → 阶段二：可测试性 → 阶段三：健壮性 → 阶段四：可维护性
```

每个阶段覆盖核心模块，目标明确、效果可见。

## 模块优先级

核心逻辑优先：

1. `pkg/trust-tunnel-agent/session/` - 会话实现
2. `pkg/trust-tunnel-agent/backend/` - 后端处理
3. `pkg/trust-tunnel-agent/sidecar/` - Sidecar 管理
4. `pkg/trust-tunnel-agent/auth/` - 认证模块
5. `pkg/common/` - 公共工具
6. `pkg/trust-tunnel-client/` - 客户端

---

## 阶段一：可读性提升

### 目标

让代码易读、易理解，建立统一风格。

### 改进点

| 改进项 | 说明 |
|--------|------|
| 命名规范化 | 统一变量、函数、常量命名风格，遵循 Go 命名惯例 |
| 注释补充 | 为导出函数添加文档注释，复杂逻辑添加说明 |
| 代码格式化 | 统一使用 gofmt、goimports，配置 golangci-lint |
| 函数拆分 | 过长函数（>50行）拆分成小函数 |
| 魔法数字 | 提取为命名常量 |

### 涉及模块

- `pkg/trust-tunnel-agent/session/`
- `pkg/trust-tunnel-agent/backend/`
- `pkg/trust-tunnel-agent/sidecar/`

### 产出

3-5 个小 PR，每个聚焦一个模块的可读性。

### 示例

**命名改进**：

```go
// Before
var d *docker.Client
func (s *session) do(ctx context.Context) error {}

// After
var dockerClient *docker.Client
func (s *session) executeCommand(ctx context.Context) error {}
```

**函数拆分**：

```go
// Before: 一个 100 行的函数处理所有逻辑

// After: 拆分为多个职责单一的函数
func (s *DockerSession) Start(ctx context.Context) error {
    if err := s.validate(); err != nil {
        return err
    }
    if err := s.createContainer(); err != nil {
        return err
    }
    return s.attachContainer()
}
```

---

## 阶段二：可测试性提升

### 目标

让代码易于测试，为后续重构提供安全网。

### 改进点

| 改进项 | 说明 |
|--------|------|
| 依赖注入 | 将硬编码依赖改为接口注入 |
| 接口抽象 | 为外部依赖定义 mock 接口 |
| 工厂模式优化 | 统一各类型 session 的创建方式 |
| 全局变量消除 | 将全局状态改为结构体字段传递 |
| 单元测试补充 | 为核心函数编写单元测试 |

### 涉及模块

- `pkg/trust-tunnel-agent/session/` - 抽象容器运行时接口
- `pkg/trust-tunnel-agent/backend/` - 注入 session 工厂
- `pkg/trust-tunnel-agent/sidecar/` - 注入 Docker 客户端

### 产出

4-6 个 PR，每个包含接口抽象 + 对应测试用例。

### 目标覆盖率

核心模块测试覆盖率 > 60%

### 示例

**接口抽象**：

```go
// Before: 直接依赖 Docker 客户端
type DockerSession struct {
    client *docker.Client
}

// After: 通过接口依赖
type ContainerRuntime interface {
    CreateContainer(ctx context.Context, config *ContainerConfig) (string, error)
    AttachContainer(ctx context.Context, id string) (io.ReadCloser, io.WriteCloser, error)
    RemoveContainer(ctx context.Context, id string) error
}

type DockerSession struct {
    runtime ContainerRuntime
}
```

**测试用例**：

```go
type mockContainerRuntime struct {
    createErr error
}

func (m *mockContainerRuntime) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
    return "container-id", m.createErr
}

func TestDockerSession_Start_Success(t *testing.T) {
    runtime := &mockContainerRuntime{}
    session := &DockerSession{runtime: runtime}

    err := session.Start(context.Background())
    assert.NoError(t, err)
}
```

---

## 阶段三：健壮性提升

### 目标

让代码更可靠，减少运行时错误。

### 改进点

| 改进项 | 说明 |
|--------|------|
| 错误处理统一 | 使用自定义错误类型，统一错误包装方式 |
| 错误信息完善 | 添加上下文信息，便于定位问题 |
| 边界条件处理 | 检查 nil、空字符串、数组越界等 |
| 资源管理 | 确保文件句柄、连接、进程正确关闭 |
| 日志规范 | 统一日志级别使用，关键路径添加日志 |
| panic 恢复 | 在 goroutine 入口添加 recover |

### 涉及模块

- `pkg/trust-tunnel-agent/session/docker.go` - 容器操作错误处理
- `pkg/trust-tunnel-agent/session/nsenter.go` - 物理机操作错误处理
- `pkg/trust-tunnel-agent/backend/handler.go` - WebSocket 处理错误
- `pkg/trust-tunnel-agent/sidecar/sidecar.go` - Sidecar 管理错误

### 产出

3-4 个 PR，每个聚焦一个错误处理场景。

### 示例

**错误处理改进**：

```go
// Before
if err != nil {
    return err
}

// After
if err != nil {
    return fmt.Errorf("failed to create container %q: %w", containerID, err)
}
```

**自定义错误类型**：

```go
type SessionError struct {
    SessionType string
    Operation   string
    Err         error
}

func (e *SessionError) Error() string {
    return fmt.Sprintf("[%s] %s failed: %v", e.SessionType, e.Operation, e.Err)
}

func (e *SessionError) Unwrap() error {
    return e.Err
}
```

**资源管理**：

```go
func (s *DockerSession) Start(ctx context.Context) (err error) {
    conn, err := s.runtime.AttachContainer(ctx, s.containerID)
    if err != nil {
        return err
    }
    defer func() {
        if closeErr := conn.Close(); closeErr != nil && err == nil {
            err = closeErr
        }
    }()
    // ...
}
```

---

## 阶段四：可维护性提升

### 目标

让代码易于修改和扩展，降低维护成本。

### 改进点

| 改进项 | 说明 |
|--------|------|
| 重复代码消除 | 提取公共函数 |
| 配置管理优化 | 统一配置结构，支持默认值和校验 |
| 常量集中管理 | 将散落的常量集中到 constants 文件 |
| 接口解耦 | 减少模块间直接依赖 |
| 代码分层 | 明确业务逻辑层、数据访问层、协议层边界 |
| TODO 清理 | 处理遗留的 TODO/FIXME 注释 |

### 涉及模块

- `pkg/trust-tunnel-agent/session/` - 提取公共 session 逻辑
- `pkg/trust-tunnel-agent/backend/` - 与 session 层解耦
- `pkg/common/` - 扩展公共工具函数
- `cmd/trust-tunnel-agent/app/` - 配置管理优化

### 产出

4-5 个 PR，重构 + 测试同步更新。

### 示例

**公共逻辑提取**：

```go
// pkg/trust-tunnel-agent/session/base.go

type BaseSession struct {
    id        string
    createdAt time.Time
    logger    *logrus.Entry
}

func (b *BaseSession) ID() string { return b.id }

func (b *BaseSession) logOperation(op string, fields logrus.Fields) {
    entry := b.logger.WithField("operation", op)
    if fields != nil {
        entry = entry.WithFields(fields)
    }
    entry.Info()
}
```

**常量集中管理**：

```go
// pkg/common/constants/session.go

const (
    SessionTypeHost      = "host"
    SessionTypeContainer = "container"

    DefaultCPULimit    = "0.5"
    DefaultMemoryLimit = "512M"
    MaxSidecarCount    = 150
)
```

---

## 整体 PR 规划

| 阶段 | PR 数量 | 改动范围 |
|------|---------|----------|
| 阶段一：可读性 | 3-5 个 | 纯重构，无功能变更 |
| 阶段二：可测试性 | 4-6 个 | 接口抽象 + 测试用例 |
| 阶段三：健壮性 | 3-4 个 | 错误处理 + 日志 |
| 阶段四：可维护性 | 4-5 个 | 代码整理 + 解耦 |

**总计**：14-20 个渐进式 PR

**阶段依赖**：

```
阶段一 ─→ 阶段二 ─→ 阶段三 ─→ 阶段四
```

每个阶段内的 PR 可并行或顺序提交，阶段间建议顺序执行。

---

## 验收标准

- [ ] 所有现有功能正常运行
- [ ] 无对外接口变更
- [ ] 核心模块测试覆盖率 > 60%
- [ ] golangci-lint 检查通过
- [ ] 端到端测试通过

---

## 开发工具配置

建议在项目根目录添加 `.golangci.yml`：

```yaml
run:
  timeout: 5m
  modules-download-mode: vendor

linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - ineffassign
    - typecheck
    - gosimple
    - goconst
    - gocyclo
    - dupl

linters-settings:
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3
```

---

## 附录：关键文件清单

### 核心模块文件

| 文件 | 作用 | 优先级 |
|------|------|--------|
| `pkg/trust-tunnel-agent/session/session.go` | 会话接口定义 | P0 |
| `pkg/trust-tunnel-agent/session/docker.go` | Docker 会话实现 | P0 |
| `pkg/trust-tunnel-agent/session/containerd.go` | Containerd 会话实现 | P0 |
| `pkg/trust-tunnel-agent/session/nsenter.go` | 物理机会话实现 | P0 |
| `pkg/trust-tunnel-agent/backend/handler.go` | WebSocket 处理器 | P0 |
| `pkg/trust-tunnel-agent/sidecar/sidecar.go` | Sidecar 管理 | P1 |
| `pkg/trust-tunnel-agent/auth/interface.go` | 认证接口 | P2 |