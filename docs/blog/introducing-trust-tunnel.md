# 当 SSH 遇上容器：我们为什么需要 Trust-Tunnel

> 蚂蚁集团开源的安全隧道工具，让你像 SSH 一样访问远程容器，但更安全、更灵活。已在蚂蚁内部大规模生产验证。

## 问题：容器时代的远程访问困境

你有没有遇到过这些场景？

- 生产环境出了问题，需要进入容器排查，但容器所在节点没有开放 SSH
- 需要批量登录上百个容器执行命令，却受限于跳板机的连接数
- 运维人员直接 `docker exec` 进入容器，执行了高危命令，事后无法追溯
- 容器网络隔离严格，外部无法直接访问，只能通过宿主机中转

传统 SSH 是为物理机时代设计的。面对 Kubernetes 集群中成千上万的容器，它显得力不从心：

| 痛点 | SSH 的局限 |
|------|-----------|
| 容器不运行 sshd | 每个容器装 SSH？镜像体积爆炸 |
| 网络不可达 | 容器网络隔离，SSH 无法穿透 |
| 权限管控 | SSH key 散落各处，难以统一管理 |
| 操作审计 | `docker exec` 不留痕 |
| 资源隔离 | 排查命令跑满 CPU，影响业务容器 |

## Trust-Tunnel：来自蚂蚁集团的安全隧道

[Trust-Tunnel](https://github.com/antgroup/trust-tunnel) 是蚂蚁集团开源的安全隧道工具，**脱胎于蚂蚁内部大规模容器运维实践**。在蚂蚁，数十万容器实例分布在全球多个机房，运维人员每天需要高频访问容器进行排障、巡检、应急响应。Trust-Tunnel 正是在这样的场景中被锻造出来的。

它通过 WebSocket 建立加密通道，让你像使用 SSH 一样访问远程容器和物理主机——但多了沙箱隔离、权限管控和完整审计。整个过程不需要在目标容器中安装 Agent 或 SSH 服务。

```
┌──────────┐        WebSocket/TLS        ┌──────────┐
│  Client  │ ◄──────────────────────────► │  Agent   │
│  (CLI)   │                              │ (节点守护)│
└──────────┘                              └────┬─────┘
                                               │
                              ┌─────────────────┼─────────────────┐
                              ▼                 ▼                  ▼
                        ┌──────────┐     ┌──────────┐      ┌──────────┐
                        │ 容器      │     │ 容器      │      │ 物理主机  │
                        │ (sidecar) │     │ (exec)   │      │ (nsenter)│
                        └──────────┘     └──────────┘      └──────────┘
```

**核心理念**：Agent 作为 DaemonSet 部署在每个节点，Client 通过 WebSocket 连接 Agent，Agent 负责在目标容器/主机中建立会话。整个过程不需要在容器里安装任何东西。

## 30 秒快速体验

```bash
# 1. 构建
git clone https://github.com/antgroup/trust-tunnel.git
cd trust-tunnel
make images && make trust-tunnel-client

# 2. 启动 Agent（本地 Docker 环境）
docker run -d --name trust-tunnel-agent \
  --privileged --pid=host --network=host \
  -v /var/run/docker.sock:/var/run-mount/docker.sock \
  -v /:/rootfs:rslave \
  trust-tunnel-agent:latest

# 3. 访问物理主机
./out/trust-tunnel-client -o 127.0.0.1 sh -c "hostname"

# 4. 访问容器（交互式）
CONTAINER_ID=$(docker ps -q --filter "name=your-app" | head -1)
./out/trust-tunnel-client -it -o 127.0.0.1 --type container --cid $CONTAINER_ID sh -c "/bin/bash"
```

没有 SSH key，没有密码，没有在容器里安装任何东西——Agent 替你搞定了一切。

## 四大核心特性

### 1. 沙箱隔离：排查问题不影响业务

这是 Trust-Tunnel 最独特的设计。开启 Clean Mode（默认）时，Agent 不会直接 `docker exec` 进入业务容器，而是创建一个 **Sidecar 容器**，共享目标容器的命名空间：

```bash
# 带资源限制的沙箱访问
./out/trust-tunnel-client -o $HOST \
  --type container --cid $CID \
  --cpus 0.5 --memory 512 \
  sh -c "strace -p 1"
```

这意味着：
- 你的排查命令跑在独立容器里，CPU/内存受限
- 即使你不小心 `rm -rf /`，删的是 sidecar，不是业务容器
- Sidecar 用完即销毁，不留痕迹

### 2. 会话复用：断线重连不丢失

网络抖动导致 WebSocket 断开？Trust-Tunnel 支持 **Stale Session** 机制：

- 连接断开后，Agent 不会立即销毁会话，而是保留一段时间
- Client 重连时自动恢复到之前的会话
- 长时间运行的命令不会因为网络波动而中断

### 3. 可插拔认证：对接你的权限体系

Trust-Tunnel 的认证是插件式的。你只需要实现一个接口：

```go
type Handler interface {
    VerifyAccessPermission(req *request.Info) Response
}
```

然后在 `auth/factory` 中注册，Agent 启动时通过配置文件加载。无论你用的是 LDAP、OAuth、还是自研的权限系统，都能无缝对接。

### 4. 全链路审计：谁在什么时候做了什么

每一次连接、每一条命令，Trust-Tunnel 都会记录：

- 谁连接的（认证信息）
- 连接了哪个目标（容器 ID / 主机 IP）
- 执行了什么命令（完整 stdin 审计）
- 什么时候连接/断开

配合 Prometheus 指标，你可以对整个集群的远程访问行为一目了然。

## 与现有方案的对比

| 特性 | SSH | kubectl exec | Trust-Tunnel |
|------|-----|--------------|--------------|
| 需要在容器中安装 | 需要 sshd | 不需要 | 不需要 |
| 沙箱隔离 | 无 | 无 | Sidecar 模式 |
| 资源限制 | 无 | 无 | CPU/内存限制 |
| 断线重连 | tmux/screen | 不支持 | 内置 Stale Session |
| 物理主机支持 | 原生 | 不支持 | 支持 |
| 操作审计 | 需要额外配置 | 有限 | 内置 |
| 认证扩展 | PAM | K8s RBAC | 可插拔接口 |
| 国密支持 | 不支持 | 不支持 | NTLS (SM2/SM3/SM4) |

## 为什么蚂蚁要做这个工具

在蚂蚁集团内部，Trust-Tunnel 的前身已经服务了多年。随着云原生技术的普及，我们发现这个问题具有普遍性——不只是蚂蚁，任何规模化运营容器的团队都会面临相同的困境。于是我们决定把它开源出来。

它不是一个实验性项目，而是**从生产环境中提炼出来的工具**，设计时就考虑了：

- **金融级安全要求**：支持国密 NTLS（SM2/SM3/SM4），满足等保和密评合规
- **大规模场景**：单节点支持 150+ 并发 Sidecar，经过蚂蚁内部万级节点验证
- **混合架构**：同时覆盖容器和物理主机，统一运维入口
- **零侵入**：不需要修改业务容器镜像，不需要在容器里安装任何东西

## 适用场景

- **运维排障**：安全地进入生产容器排查问题，不影响业务
- **安全合规**：所有远程操作经过认证、审计，满足等保要求
- **多租户隔离**：不同团队通过认证插件获得不同访问权限
- **混合环境**：同时管理容器和物理主机，统一访问方式
- **国密场景**：支持 NTLS（基于铜锁/Tongsuo），满足国产密码合规

## 生产部署建议

推荐使用 Helm 将 Agent 部署为 DaemonSet：

```bash
helm install trust-tunnel-agent ./charts/trust-tunnel-agent
```

Agent 以特权模式运行在每个节点上，监听 5006 端口。Client 可以从任意能访问该端口的网络发起连接。

对于生产环境，建议：
1. 启用 TLS/NTLS 加密
2. 配置认证插件，对接内部权限系统
3. 设置 Sidecar 容器数量上限（默认 150/节点）
4. 对接日志平台收集审计日志

## 技术栈

- **语言**：Go 1.21+
- **通信**：WebSocket
- **容器运行时**：Docker SDK / containerd
- **加密**：标准 TLS / NTLS（铜锁 Tongsuo）
- **监控**：Prometheus
- **部署**：Helm Chart / DaemonSet

## 参与贡献

Trust-Tunnel 采用 Apache 2.0 协议，由蚂蚁集团开源维护。作为蚂蚁开源矩阵的一部分（同系列还有 [SOFAStack](https://github.com/sofastack)、[OceanBase](https://github.com/oceanbase/oceanbase)、[Ant Design](https://github.com/ant-design/ant-design) 等知名项目），Trust-Tunnel 致力于将蚂蚁内部的基础设施能力回馈社区。

我们欢迎各种形式的贡献：

- 提 Issue 反馈问题或建议
- 提 PR 改进代码
- 分享使用经验和最佳实践
- 在你的技术社区介绍 Trust-Tunnel

GitHub: https://github.com/antgroup/trust-tunnel

---

*Trust-Tunnel 由蚂蚁集团安全团队开源维护。如果这个项目对你有帮助，欢迎给个 Star。有问题可以在 GitHub Issues 中讨论。*
