// Copyright The TrustTunnel Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"trust-tunnel/pkg/common/logutil"
	"trust-tunnel/pkg/common/sessionutil"
	"trust-tunnel/pkg/trust-tunnel-agent/auth"
	"trust-tunnel/pkg/trust-tunnel-agent/backend/request"
	"trust-tunnel/pkg/trust-tunnel-agent/sidecar"

	_ "trust-tunnel/pkg/trust-tunnel-agent/auth/example"
	agentSession "trust-tunnel/pkg/trust-tunnel-agent/session"
	client "trust-tunnel/pkg/trust-tunnel-client"

	"github.com/containerd/containerd"
	dockerAPIClient "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var logger = logutil.GetLogger("trust-tunnel-agent")

const (
	maxWebsocketControlMsgLength = 123
)

// Config represents the configuration for the Handler.
type Config struct {
	// SessionConfig specifies the session configuration.
	SessionConfig SessionConfig

	// ContainerConfig specifies the container configuration.
	ContainerConfig agentSession.ContainerConfig

	// AuthConfig specifies the authentication configuration.
	AuthConfig auth.Config

	// SidecarConfig specifies the sidecar configuration.
	SidecarConfig sidecar.Config
}

// Handler represents a WebSocket handler for establishing sessions.
type Handler struct {
	config            *Config
	staleSessions     map[string]*StaleSession
	dockerClient      dockerAPIClient.CommonAPIClient
	containerdClient  *containerd.Client
	authHandler       auth.Handler
	lock              sync.Mutex
	currentSidecarNum int
}

// NewHandler creates a new Handler with the given configuration.
func NewHandler(c *Config) (*Handler, error) {
	h := &Handler{
		config:        c,
		staleSessions: make(map[string]*StaleSession),
	}
	// Create a container client based on the container runtime.
	if h.config.ContainerConfig.ContainerRuntime == agentSession.Docker {
		dockerClient, err := sessionutil.CreateDockerClient(c.ContainerConfig.Endpoint, c.ContainerConfig.DockerAPIVersion)
		if err != nil {
			logger.Errorf("create container API client error: %s", err.Error())
		} else {
			h.dockerClient = dockerClient
		}
	} else {
		containerdClient, err := containerd.New(c.ContainerConfig.Endpoint)
		if err != nil {
			logger.Errorf("create containerd API client error: %s", err.Error())
		} else {
			h.containerdClient = containerdClient
		}
	}

	// Init the authHandler.
	var authHandler auth.Handler

	var err error

	if c.AuthConfig.Name != "" {
		authHandler, err = auth.CreateAuthHandlerFromConfig(c.AuthConfig)
		if err != nil {
			log.Fatalf("Failed to create authHandler: %v", err)
		}
	}

	h.authHandler = authHandler

	// Pull the sidecar image during booting.
	err := sidecar.Init(c.ContainerConfig.Endpoint, c.SidecarConfig.Image, c.SidecarConfig.ImageHubAuth, h.dockerClient)
	if err != nil {
		logger.Errorf("init sidecar with image %s error: %v, ignore it", c.SidecarConfig.Image, err)
	}
	// Clean legacy sidecar container periodically.
	go sidecar.CleanLegacyContainerPeriodically(h.dockerClient)

	// Delay release stale sessions.
	go h.delayReleaseSession()

	return h, nil
}

var upgrader = websocket.Upgrader{}

// Handle handles the incoming HTTP request and establishes a new session.
func (handler *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	// Create a logger for the incoming request.
	requestLogger := logger.WithField("request_from", r.RemoteAddr)

	// Get the request information from the incoming request.
	requestInfo, err := request.GetRequestInfo(r)
	if err != nil {
		requestLogger.Warnln("Request invalid: ", err)

		return
	}

	// Log the request information.
	requestLogger.Infoln("Request info: ", requestInfo)

	// Check if the user has the permission the access the target.
	if handler.authHandler != nil {
		authResult := handler.authHandler.VerifyAccessPermission(requestInfo)
		if authResult.Code != auth.Success {
			logger.Errorf("authorization failed:%v", authResult)

			return
		}
	}

	// Construct request info to audit log.
	constructAuditInfo(requestInfo)

	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		requestLogger.Warnln("Websocket upgrade error: ", err)

		return
	}
	defer conn.Close()

	// Create a session configuration from the request information.
	sessConf := &agentSession.Config{
		TargetType:       requestInfo.TargetType,
		UserName:         requestInfo.UserName,
		LoginName:        requestInfo.LoginName,
		LoginGroup:       requestInfo.LoginGroup,
		ContainerID:      requestInfo.ContainerID,
		Cmd:              requestInfo.Cmd,
		Tty:              requestInfo.Tty,
		Interactive:      requestInfo.Interactive,
		PhysTunnel:       handler.config.SessionConfig.PhysTunnel,
		SidecarImage:     handler.config.SidecarConfig.Image,
		ImageHubAuth:     handler.config.SidecarConfig.ImageHubAuth,
		Cpus:             requestInfo.Cpus,
		MemoryMB:         requestInfo.MemoryMB,
		DisableCleanMode: requestInfo.DisableCleanMode,
		RootfsPrefix:     handler.config.ContainerConfig.RootfsPrefix,
	}

	var (
		sess   agentSession.Session
		sessID = requestInfo.SessionID
	)

	// Find un-released sessions from list, and reuse it if exists.
	handler.lock.Lock()
	if staleSess, ok := handler.staleSessions[sessID]; ok && sessID != "" && requestInfo.UserName == staleSess.userName {
		sess = staleSess.sess
		// Remove stale session from list.
		delete(handler.staleSessions, sessID)
		requestLogger.Infof("reuse stale session %s", sessID)
	}
	handler.lock.Unlock()

	// If session ID is not found in stale sessions, create a new session.
	if sessID == "" {
		sessID = time.Now().Format("20060102150405")
	}

	// Create a logger for the session.
	requestLogger = requestLogger.WithField("session_id", sessID)

	// Check if the session needs to attach a sidecar to the container.
	var isSidecarSession bool

	// Session ID not found in stale sessions, create a new session.
	if sess == nil {
		if sessConf.TargetType == client.TargetContainer {
			isSidecarSession, err = handler.containerPreCheck(sessConf, handler.config.ContainerConfig.ContainerRuntime)
			if err != nil {
				errMsg := sessionutil.WrapErrorWithCode(sessionutil.WrapContainerError(err.Error(), sessConf.ContainerID))
				logger.Error(errMsg)
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, truncWebsocketErrMsg("Establish session error: "+errMsg)))

				return
			}
		}

		sess, err = agentSession.EstablishSession(sessConf, handler.dockerClient, handler.containerdClient, handler.config.ContainerConfig.ContainerRuntime)
		if err != nil {
			requestLogger.Warnf("Establish session error: %v", err)
			errMsg := sessionutil.WrapErrorWithCode(err.Error())
			logger.Error(errMsg)
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, truncWebsocketErrMsg("Establish session error: "+errMsg)))

			return
		}

		if isSidecarSession {
			handler.currentSidecarNum++
		}

		requestLogger.Infoln("new session established")
	}

	// Create a new connection for the session.
	sessConn := &Connection{
		conn: conn,
		sess: sess,
		// Create a new command logger.
		cmdLogger: createCmdLogger(requestLogger, requestInfo),
		errCh:     make(chan error, 1),
		doneCh:    make(chan struct{}),
	}
	defer sessConn.cmdLogger.Destroy()

	// Start the input, output, and error processing goroutines.
	go sessConn.processRemoteInput()
	go sessConn.processLocalOutput()
	go sessConn.processLocalError()

	// Wait for an error to occur.
	err = <-sessConn.errCh

	handler.lock.Lock()
	if err != nil {
		// Client is closed abnormally.
		// Append stale session to list for delay release.
		handler.staleSessions[sessID] = &StaleSession{
			userName:         requestInfo.UserName,
			sess:             sess,
			deathClock:       time.After(handler.config.SessionConfig.DelayReleaseSessionTimeout),
			isSidecarSession: isSidecarSession,
		}

		requestLogger.Infof("reserve session %s\n", sessID)
	} else {
		// Do cleanup.
		err = handler.releaseSession(sessID, sess)
		if err == nil && isSidecarSession {
			handler.currentSidecarNum--
		}
	}
	handler.lock.Unlock()

	if err != nil {
		requestLogger.Infoln("session disconnected with err: ", err)
	} else {
		requestLogger.Infoln("session disconnected")
	}
}

// containerPreCheck does some pre-checks before establishing the session:
// 1. check if the container runtime is ready.
// 2. check if the current sidecar container num exceeds the limit.
func (handler *Handler) containerPreCheck(sessConf *agentSession.Config, runtime agentSession.ContainerRuntime) (bool, error) {
	var isContainerSidecarSession bool

	err := handler.checkContainerRuntime(sessConf, runtime)
	if err != nil {
		return isContainerSidecarSession, err
	}

	return handler.checkSidecarNum(sessConf, runtime)
}

// checkContainerRuntime checks if the container runtime is ready.
func (handler *Handler) checkContainerRuntime(sessConf *agentSession.Config, runtime agentSession.ContainerRuntime) error {
	var err error
	// In case of when trust-tunnel-agent starts,the container daemon is not ready,but after some time the container daemon is ready again,
	if sessConf.TargetType == client.TargetContainer && (runtime == agentSession.Docker) && handler.dockerClient == nil {
		handler.dockerClient, err = sessionutil.CreateDockerClient(handler.config.ContainerConfig.Endpoint, handler.config.ContainerConfig.DockerAPIVersion)
		if err != nil {
			return err
		}
	} else if runtime == agentSession.Containerd && handler.containerdClient == nil {
		handler.containerdClient, err = containerd.New(handler.config.ContainerConfig.Endpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkSidecarNum checks if current sidecar num exceeds the limit.
func (handler *Handler) checkSidecarNum(sessConf *agentSession.Config, runtime agentSession.ContainerRuntime) (bool, error) {
	var isContainerSidecarSession bool

	if runtime == agentSession.Docker {
		if !sessConf.DisableCleanMode {
			isContainerSidecarSession = true
			// if current sidecar num exceed the limit,just return error.
			if handler.currentSidecarNum >= handler.config.SidecarConfig.Limit {
				return isContainerSidecarSession, fmt.Errorf("current sidecar num exceed the limit: %d,%d ", handler.currentSidecarNum, handler.config.SidecarConfig.Limit)
			}
		}
	}

	return false, nil
}

// createCmdLogger creates a new CmdLogger with the given logger and request information.
func createCmdLogger(logger *logrus.Entry, req *request.Info) *logutil.CmdLogger {
	fields := logrus.Fields{
		"session_id":         req.SessionID,
		"user_name":          req.UserName,
		"login_name":         req.LoginName,
		"target_type":        req.TargetType,
		"pod":                req.PodName,
		"container_id":       req.ContainerID,
		"container_name":     req.ContainerName,
		"ip":                 req.IPAddress,
		"cpus":               req.Cpus,
		"memoryMB":           req.MemoryMB,
		"disable_clean_mode": req.DisableCleanMode,
	}
	logger = logger.WithFields(fields)
	cmdLogger := logutil.NewCmdLogger(logger)
	logger.Debugf("InitCmd: %#v", req.Cmd)

	return cmdLogger
}

// According to websocket rfc protocol,RFC6455,
// All control frames MUST have a payload length of 125 bytes or fewer and MUST NOT be fragmented.
// Two bytes reserved for the close code,so we have 123 bytes left for the error message.
func truncWebsocketErrMsg(errMsg string) string {
	if len(errMsg) > maxWebsocketControlMsgLength {
		errMsg = errMsg[0:maxWebsocketControlMsgLength]
	}

	return errMsg
}
