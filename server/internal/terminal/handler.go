// Package terminal 服务端远程终端中继：浏览器 ↔ GRP Stream ↔ Agent PTY。
package terminal

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"gateway-manager/pkg/grpv1"
	"gateway-manager/pkg/logger"
	"gateway-manager/server/internal/api"
	"gateway-manager/server/internal/device"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(*http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// ctrlMsg 浏览器侧控制消息（与终端数据共用同一 WebSocket，按 type 区分）
type ctrlMsg struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
	Shell    string `json:"shell"`
	Rows     uint32 `json:"rows"`
	Cols     uint32 `json:"cols"`
	Message  string `json:"message"`
}

// Register 注册浏览器终端 WebSocket 路由
func Register(r gin.IRoutes) {
	r.GET("/api/terminal/ws", HandleWS)
}

// HandleWS 浏览器终端入口
func HandleWS(c *gin.Context) {
	token := c.Query("token")
	if _, err := api.AuthenticateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// 读取 open 控制消息
	_, data, err := ws.ReadMessage()
	if err != nil {
		return
	}
	var open ctrlMsg
	if err := json.Unmarshal(data, &open); err != nil || open.Type != "open" || open.DeviceID == "" {
		_ = writeCtrl(ws, "error", "无效的打开请求")
		return
	}
	if open.Cols == 0 {
		open.Cols = 80
	}
	if open.Rows == 0 {
		open.Rows = 24
	}
	if open.Shell == "" {
		open.Shell = "/bin/sh"
	}

	conn, ok := device.Default.Connection(open.DeviceID)
	if !ok {
		_ = writeCtrl(ws, "error", "设备不在线")
		return
	}

	ctx := c.Request.Context()
	stream, err := conn.OpenStream(ctx, grpv1.StreamOptions{ContentType: "terminal"})
	if err != nil {
		_ = writeCtrl(ws, "error", "打开流失败: "+err.Error())
		return
	}
	defer stream.Close()

	sessionID := uuid.NewString()
	reqEnv := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeTerminalOpen, open.DeviceID)
	reqEnv.Payload, _ = grpv1.MarshalPayload(grpv1.TerminalOpen{
		SessionID: sessionID,
		StreamID:  stream.ID(),
		Shell:     open.Shell,
		Rows:      open.Rows,
		Cols:      open.Cols,
	})

	resp, err := conn.Request(ctx, reqEnv, 10*time.Second)
	if err != nil {
		_ = writeCtrl(ws, "error", "打开终端失败: "+err.Error())
		return
	}
	var result grpv1.TerminalOpenResult
	_ = grpv1.UnmarshalPayload(resp.Payload, &result)
	if !result.Success {
		_ = writeCtrl(ws, "error", result.Message)
		return
	}

	logger.Info("终端会话建立", "device", open.DeviceID, "session", sessionID, "stream", stream.ID())

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() { once.Do(func() { close(done) }) }

	// Agent -> 浏览器
	go func() {
		defer closeDone()
		buf := make([]byte, 4096)
		for {
			n, err := stream.Read(ctx, buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.TextMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// 浏览器 -> Agent
	go func() {
		defer closeDone()
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if msg, ok := parseCtrl(data); ok {
				if msg.Type == "resize" {
					ev, _ := grpv1.NewEventEnvelope(grpv1.TypeTerminalResize, open.DeviceID, grpv1.TerminalResize{
						SessionID: sessionID, Rows: msg.Rows, Cols: msg.Cols,
					})
					_ = conn.Send(ctx, ev)
				}
				continue
			}
			if _, err := stream.Write(ctx, data); err != nil {
				return
			}
		}
	}()

	<-done

	// 清理：关闭流并通知 Agent 结束会话
	_ = stream.Close()
	ev, _ := grpv1.NewEventEnvelope(grpv1.TypeTerminalClose, open.DeviceID, grpv1.TerminalClose{SessionID: sessionID})
	_ = conn.Send(context.Background(), ev)
}

func parseCtrl(data []byte) (ctrlMsg, bool) {
	if len(data) == 0 || data[0] != '{' {
		return ctrlMsg{}, false
	}
	var m ctrlMsg
	if err := json.Unmarshal(data, &m); err != nil {
		return ctrlMsg{}, false
	}
	if m.Type == "" {
		return ctrlMsg{}, false
	}
	return m, true
}

func writeCtrl(ws *websocket.Conn, typ, message string) error {
	b, _ := json.Marshal(ctrlMsg{Type: typ, Message: message})
	return ws.WriteMessage(websocket.TextMessage, b)
}