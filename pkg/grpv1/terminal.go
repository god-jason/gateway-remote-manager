package grpv1

// 终端消息类型
const (
	TypeTerminalOpen  = "terminal.session.open"
	TypeTerminalInput = "terminal.input"
	TypeTerminalOutput = "terminal.output"
	TypeTerminalResize = "terminal.resize"
	TypeTerminalClose  = "terminal.session.close"
)

// TerminalOpen 打开终端会话（由 Server 发起，携带关联的 Stream ID）
type TerminalOpen struct {
	SessionID        string `json:"session_id"`
	StreamID         uint32 `json:"stream_id"`
	Shell            string `json:"shell"`
	User             string `json:"user,omitempty"`
	Rows             uint32 `json:"rows"`
	Cols             uint32 `json:"cols"`
	WorkingDirectory string `json:"working_directory,omitempty"`
}

// TerminalOpenResult 打开结果
type TerminalOpenResult struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

// TerminalInput 终端输入（通常走 Stream 传输，保留用于控制场景）
type TerminalInput struct {
	SessionID string `json:"session_id"`
	Data      []byte `json:"data,omitempty"`
}

// TerminalOutput 终端输出（通常走 Stream 传输）
type TerminalOutput struct {
	SessionID string `json:"session_id"`
	Data      []byte `json:"data,omitempty"`
}

// TerminalResize 调整终端窗口
type TerminalResize struct {
	SessionID string `json:"session_id"`
	Rows      uint32 `json:"rows"`
	Cols      uint32 `json:"cols"`
}

// TerminalClose 关闭终端会话
type TerminalClose struct {
	SessionID string `json:"session_id"`
	Reason    uint32 `json:"reason,omitempty"`
}