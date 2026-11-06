package connection

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"gateway-manager/pkg/config"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/pkg/logger"
	"gateway-manager/server/internal/device"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(*http.Request) bool { return true }, // TODO 生产环境按来源限制
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// Register 注册 Agent WebSocket 路由
func Register(r gin.IRoutes) {
	r.GET("/agent/ws", HandleAgentWS)
}

// HandleAgentWS Agent 长连接入口
func HandleAgentWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn := newConn(ws, uuid.NewString())
	defer conn.Close()

	handleConn(c.Request.Context(), conn, ws)
}

// handleConn 一条连接的完整生命周期：握手 → 上线 → 循环分发
func handleConn(ctx context.Context, conn *Conn, ws *websocket.Conn) {
	_ = ws.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 1. hello
	env, err := conn.readEnvelope()
	if err != nil || env.Type != grpv1.TypeHandshakeHello {
		if err != nil {
			logger.Debug("读取 hello 失败", "error", err)
		}
		return
	}
	var hello grpv1.Hello
	if err := grpv1.UnmarshalPayload(env.Payload, &hello); err != nil || hello.DeviceID == "" {
		return
	}
	conn.SetDeviceID(hello.DeviceID)

	// 2. 握手认证
	if err := doHandshake(conn, hello); err != nil {
		logger.Warn("设备握手失败", "device", hello.DeviceID, "error", err)
		return
	}

	// 3. 上线
	device.Default.Register(conn)
	defer device.Default.Unregister(conn)
	defer func() { _ = device.MarkOffline(hello.DeviceID) }()
	_ = device.MarkOnline(hello.DeviceID, hello.AgentVersion, hello.OS, hello.Arch)

	_ = ws.SetReadDeadline(time.Time{})
	logger.Info("设备已连接", "device", conn.deviceID, "conn", conn.id)

	// 4. 循环分发
	conn.loop(ctx)
}

// doHandshake hello -> challenge -> auth -> auth_result
func doHandshake(conn *Conn, hello grpv1.Hello) error {
	allowAuto := config.GetBool("auth", "allow_auto_register")
	if err := device.EnsureDevice(hello, allowAuto); err != nil {
		return err
	}

	challenge := grpv1.Challenge{
		ChallengeID: uuid.NewString(),
		Nonce:       uuid.NewString(),
		ExpiresAt:   time.Now().Add(30 * time.Second).UnixMilli(),
	}
	challengeEnv, err := grpv1.NewEventEnvelope(grpv1.TypeHandshakeChallenge, hello.DeviceID, challenge)
	if err != nil {
		return err
	}
	if err := conn.Send(context.Background(), challengeEnv); err != nil {
		return err
	}

	authEnv, err := conn.readEnvelope()
	if err != nil {
		return err
	}
	if authEnv.Type != grpv1.TypeHandshakeAuth {
		return grpv1.NewError(grpv1.ErrInvalidMessage, "期望 auth 消息")
	}
	var auth grpv1.Authenticate
	if err := grpv1.UnmarshalPayload(authEnv.Payload, &auth); err != nil {
		return err
	}

	key, found, err := device.FindCredential(hello.DeviceID)
	if err != nil {
		return err
	}
	if !found {
		_ = sendAuthResult(conn, hello.DeviceID, false, "设备无凭据")
		return grpv1.NewError(grpv1.ErrDeviceAuthFailed, "设备无凭据")
	}
	if !grpv1.VerifyChallenge(auth.Signature, key, challenge.Nonce) {
		_ = sendAuthResult(conn, hello.DeviceID, false, "签名校验失败")
		return grpv1.NewError(grpv1.ErrDeviceAuthFailed, "签名校验失败")
	}

	return sendAuthResult(conn, hello.DeviceID, true, "")
}

func sendAuthResult(conn *Conn, deviceID string, ok bool, msg string) error {
	res := grpv1.AuthenticateResult{
		Success:           ok,
		ConnectionID:      conn.id,
		ServerNodeID:      "node-1",
		HeartbeatInterval: uint32(config.GetInt("server", "heartbeat_interval")),
		HeartbeatTimeout:  uint32(config.GetInt("server", "heartbeat_timeout")),
		Message:           msg,
	}
	env, err := grpv1.NewEventEnvelope(grpv1.TypeHandshakeAuthResult, deviceID, res)
	if err != nil {
		return err
	}
	return conn.Send(context.Background(), env)
}