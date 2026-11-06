// Package core Agent 运行时：连接、握手认证、心跳与状态上报、自动重连。
package core

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"gateway-manager/agent/internal/filesystem"
	"gateway-manager/agent/internal/identity"
	"gateway-manager/agent/internal/system"
	"gateway-manager/agent/internal/terminal"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/pkg/logger"
)

// AgentVersion 当前 Agent 版本
const AgentVersion = "0.1.0"

// Config Agent 配置
type Config struct {
	Server            string        // 服务器 WSS/WS 地址
	HeartbeatInterval time.Duration // 心跳间隔
	ReportInterval    time.Duration // 状态上报间隔
	IdentityFile      string        // 身份文件路径
	DeviceID          string        // 可选：强制指定设备 ID（覆盖/初始化身份）
	DeviceKey         string        // 可选：强制指定设备密钥
}

// Agent 网关运行时
type Agent struct {
	cfg       Config
	identity  *identity.Identity
	client    *grpv1.WebsocketClient
	startTime time.Time
	seq       uint64
}

// New 创建 Agent
func New(cfg Config) *Agent {
	return &Agent{
		cfg:       cfg,
		startTime: time.Now(),
	}
}

const (
	maxBackoff = 60 * time.Second
	baseBackoff = time.Second
)

// Run 主循环：连接、断线退避重连
func (a *Agent) Run(ctx context.Context) error {
	id, err := identity.Load(a.cfg.IdentityFile)
	if err != nil {
		return err
	}
	// 命令行覆盖
	if a.cfg.DeviceID != "" {
		id.DeviceID = a.cfg.DeviceID
	}
	if a.cfg.DeviceKey != "" {
		id.DeviceKey = a.cfg.DeviceKey
	}
	if err := id.Save(a.cfg.IdentityFile); err != nil {
		logger.Warn("身份持久化失败", "error", err)
	}
	a.identity = id

	var backoff = baseBackoff
	for ctx.Err() == nil {
		if err := a.runOnce(ctx); err != nil && ctx.Err() == nil {
			logger.Warn("连接中断", "error", err)
		}
		if ctx.Err() != nil {
			break
		}

		// 指数退避 + 随机抖动
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		jitter := time.Duration(rand.Int63n(int64(backoff)/2 + 1))
		delay := backoff + jitter
		backoff *= 2

		logger.Info("等待重连", "delay", delay.String())
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
	return nil
}

// runOnce 建立一次连接并完成握手、心跳、上报，直到断线
func (a *Agent) runOnce(ctx context.Context) error {
	client := grpv1.NewWebsocketClient(a.cfg.Server, nil)
	client.SetDeviceID(a.identity.DeviceID)
	a.client = client

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := client.Connect(connCtx); err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer client.Close()

	if err := a.handshake(connCtx, client); err != nil {
		return err
	}
	logger.Info("设备上线", "device_id", a.identity.DeviceID)

	if err := a.sendCapabilities(connCtx, client); err != nil {
		return err
	}

	// 远程终端（每个连接独立的会话管理器）
	termMgr := terminal.NewManager()
	defer termMgr.CloseAll()
	a.registerTerminal(client, termMgr)

	// 文件传输
	a.registerFiles(client)

	done := make(chan struct{})
	var once sync.Once
	client.OnDisconnect(func() { once.Do(func() { close(done) }) })

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.heartbeatLoop(connCtx, client) }()
	go func() { defer wg.Done(); a.statusLoop(connCtx, client) }()

	select {
	case <-done:
	case <-ctx.Done():
	}
	cancel()
	wg.Wait()
	return nil
}

// handshake 连接握手：hello -> challenge -> auth -> auth_result
func (a *Agent) handshake(ctx context.Context, client *grpv1.WebsocketClient) error {
	challengeCh := make(chan *grpv1.Challenge, 1)
	authResultCh := make(chan *grpv1.AuthenticateResult, 1)

	client.RegisterHandler(grpv1.TypeHandshakeChallenge, func(ctx context.Context, env *grpv1.Envelope) (*grpv1.Envelope, error) {
		var ch grpv1.Challenge
		if err := grpv1.UnmarshalPayload(env.Payload, &ch); err != nil {
			return nil, err
		}
		challengeCh <- &ch
		return nil, nil
	})
	client.RegisterHandler(grpv1.TypeHandshakeAuthResult, func(ctx context.Context, env *grpv1.Envelope) (*grpv1.Envelope, error) {
		var r grpv1.AuthenticateResult
		if err := grpv1.UnmarshalPayload(env.Payload, &r); err != nil {
			return nil, err
		}
		authResultCh <- &r
		return nil, nil
	})

	// 1. hello（携带 device_key，便于服务端在凭据缺失时自动恢复注册。
	//     第一阶段密钥靠 TLS 保护，后续阶段替换为 ed25519 / mTLS 设备证书）
	hello := grpv1.Hello{
		DeviceID:        a.identity.DeviceID,
		DeviceKey:       a.identity.DeviceKey,
		AgentVersion:    AgentVersion,
		ProtocolVersion: fmt.Sprintf("%d", grpv1.ProtocolVersion),
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		Nonce:           uuid.NewString(),
	}
	helloEnv, err := grpv1.NewEventEnvelope(grpv1.TypeHandshakeHello, a.identity.DeviceID, hello)
	if err != nil {
		return err
	}
	if err := client.Send(ctx, helloEnv); err != nil {
		return err
	}

	// 2. 等待 challenge
	var challenge *grpv1.Challenge
	select {
	case challenge = <-challengeCh:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("等待握手 challenge 超时")
	}

	// 3. auth（HMAC 签名）
	auth := grpv1.Authenticate{
		DeviceID:    a.identity.DeviceID,
		ChallengeID: challenge.ChallengeID,
		Signature:   a.identity.Sign(challenge.Nonce),
	}
	authEnv, err := grpv1.NewEventEnvelope(grpv1.TypeHandshakeAuth, a.identity.DeviceID, auth)
	if err != nil {
		return err
	}
	if err := client.Send(ctx, authEnv); err != nil {
		return err
	}

	// 4. 等待 auth_result
	var result *grpv1.AuthenticateResult
	select {
	case result = <-authResultCh:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("等待认证结果超时")
	}

	if !result.Success {
		return fmt.Errorf("认证失败: %s", result.Message)
	}

	a.identity.Registered = true
	if err := a.identity.Save(a.cfg.IdentityFile); err != nil {
		logger.Warn("身份持久化失败", "error", err)
	}
	return nil
}

// sendCapabilities 上报设备能力
func (a *Agent) sendCapabilities(ctx context.Context, client *grpv1.WebsocketClient) error {
	cap := grpv1.Capabilities{
		Features:            []string{"system.monitor", "terminal"},
		Commands:            []string{grpv1.TypeSystemStatus, grpv1.TypeHeartbeat, grpv1.TypeTerminalOpen, grpv1.TypeTerminalResize, grpv1.TypeTerminalClose},
		MaxTerminalSessions: 5,
		MaxFileTransfers:    3,
		MaxFileSize:         1 << 30,
	}
	env, err := grpv1.NewEventEnvelope(grpv1.TypeDeviceCapabilities, a.identity.DeviceID, cap)
	if err != nil {
		return err
	}
	return client.Send(ctx, env)
}

// heartbeatLoop 周期性心跳
func (a *Agent) heartbeatLoop(ctx context.Context, client *grpv1.WebsocketClient) {
	ticker := time.NewTicker(a.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hb := grpv1.Heartbeat{
				AgentUptime: uint64(time.Since(a.startTime).Seconds()),
				Sequence:    atomic.AddUint64(&a.seq, 1),
				AgentState:  "running",
			}
			env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeHeartbeat, a.identity.DeviceID)
			env.Payload, _ = grpv1.MarshalPayload(hb)

			resp, err := client.Request(ctx, env, 5*time.Second)
			if err != nil {
				logger.Warn("心跳失败", "error", err)
				continue
			}
			var ack grpv1.HeartbeatAck
			if err := grpv1.UnmarshalPayload(resp.Payload, &ack); err == nil && ack.NextHeartbeatInterval > 0 {
				if d := time.Duration(ack.NextHeartbeatInterval) * time.Second; d != a.cfg.HeartbeatInterval {
					ticker.Reset(d)
				}
			}
		}
	}
}

// statusLoop 周期性状态上报
func (a *Agent) statusLoop(ctx context.Context, client *grpv1.WebsocketClient) {
	ticker := time.NewTicker(a.cfg.ReportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st := system.Collect(time.Since(a.startTime))
			env, err := grpv1.NewEventEnvelope(grpv1.TypeSystemStatus, a.identity.DeviceID, st)
			if err != nil {
				continue
			}
			if err := client.Send(ctx, env); err != nil {
				logger.Warn("状态上报失败", "error", err)
			}
		}
	}
}

// registerTerminal 注册远程终端相关处理器
func (a *Agent) registerTerminal(client *grpv1.WebsocketClient, mgr *terminal.Manager) {
	client.RegisterHandler(grpv1.TypeTerminalOpen, func(ctx context.Context, env *grpv1.Envelope) (*grpv1.Envelope, error) {
		var open grpv1.TerminalOpen
		if err := grpv1.UnmarshalPayload(env.Payload, &open); err != nil {
			return terminalResultEnvelope(false, "", err.Error()), nil
		}
		stream := client.GetStream(open.StreamID)
		if stream == nil {
			return terminalResultEnvelope(false, "", "stream not found"), nil
		}
		if err := mgr.Open(ctx, open, stream); err != nil {
			return terminalResultEnvelope(false, "", err.Error()), nil
		}
		return terminalResultEnvelope(true, open.SessionID, ""), nil
	})

	client.RegisterHandler(grpv1.TypeTerminalResize, func(ctx context.Context, env *grpv1.Envelope) (*grpv1.Envelope, error) {
		var r grpv1.TerminalResize
		if err := grpv1.UnmarshalPayload(env.Payload, &r); err == nil {
			_ = mgr.Resize(r.SessionID, uint16(r.Rows), uint16(r.Cols))
		}
		return nil, nil
	})

	client.RegisterHandler(grpv1.TypeTerminalClose, func(ctx context.Context, env *grpv1.Envelope) (*grpv1.Envelope, error) {
		var cl grpv1.TerminalClose
		if err := grpv1.UnmarshalPayload(env.Payload, &cl); err == nil {
			mgr.Close(cl.SessionID)
		}
		return nil, nil
	})
}

// terminalResultEnvelope 构造终端打开结果响应
func terminalResultEnvelope(success bool, sessionID, message string) *grpv1.Envelope {
	payload, _ := grpv1.MarshalPayload(grpv1.TerminalOpenResult{
		Success:   success,
		SessionID: sessionID,
		Message:   message,
	})
	return &grpv1.Envelope{Payload: payload}
}

// registerFiles 注册文件传输处理器
func (a *Agent) registerFiles(client *grpv1.WebsocketClient) {
	sendResult := func(r grpv1.FileTransferResult) {
		if env, err := grpv1.NewEventEnvelope(grpv1.TypeFileResult, a.identity.DeviceID, r); err == nil {
			_ = client.Send(context.Background(), env)
		}
	}

	client.RegisterHandler(grpv1.TypeFileOpen, func(ctx context.Context, env *grpv1.Envelope) (*grpv1.Envelope, error) {
		var open grpv1.FileTransferOpen
		if err := grpv1.UnmarshalPayload(env.Payload, &open); err != nil {
			return fileAckEnvelope(false, "", "", err.Error()), nil
		}
		stream := client.GetStream(open.StreamID)
		if stream == nil {
			return fileAckEnvelope(false, open.TransferID, "", "stream not found"), nil
		}

		switch open.Direction {
		case grpv1.DirectionDownload:
			info, err := os.Stat(filepath.Clean(open.Path))
			if err != nil {
				return fileAckEnvelope(false, open.TransferID, "", err.Error()), nil
			}
			if info.IsDir() {
				return fileAckEnvelope(false, open.TransferID, "", "目标为目录"), nil
			}
			go filesystem.Download(ctx, stream, open.Path, sendResult)
			return fileAckEnvelope(true, open.TransferID, info.Size(), ""), nil
		case grpv1.DirectionUpload:
			go filesystem.Upload(ctx, stream, open, sendResult)
			return fileAckEnvelope(true, open.TransferID, "", ""), nil
		default:
			return fileAckEnvelope(false, open.TransferID, "", "未知传输方向"), nil
		}
	})
}

// fileAckEnvelope 构造文件传输 open 应答
func fileAckEnvelope(success bool, transferID string, fileSize int64, message string) *grpv1.Envelope {
	payload, _ := grpv1.MarshalPayload(grpv1.FileTransferResult{
		Success:    success,
		TransferID: transferID,
		FileSize:   fileSize,
		Message:    message,
	})
	return &grpv1.Envelope{Payload: payload}
}