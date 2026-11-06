package grpv1

// 任务消息类型
const (
	TypeTaskCreate   = "task.create"
	TypeTaskCancel   = "task.cancel"
	TypeTaskResult   = "task.result"
	TypeTaskProgress = "task.progress"
)

// 任务类型
const (
	TaskTypeCommand = "command"
	TaskTypeScript  = "script"
)

// 任务终结状态（Agent 回报）
const (
	TaskStatusSuccess   = "success"
	TaskStatusFailed    = "failed"
	TaskStatusTimeout   = "timeout"
	TaskStatusCancelled = "cancelled"
)

// TaskCreate 创建任务（下发到 Agent）
type TaskCreate struct {
	TaskID         string `json:"task_id"`
	TaskType       string `json:"task_type"`
	Command        string `json:"command,omitempty"`
	Payload        string `json:"payload,omitempty"`
	TimeoutSeconds uint32 `json:"timeout_seconds,omitempty"`
	Idempotent     bool   `json:"idempotent,omitempty"`
}

// TaskCreateResult 任务创建应答
type TaskCreateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	TaskID  string `json:"task_id,omitempty"`
}

// TaskCancelRequest 取消任务
type TaskCancelRequest struct {
	TaskID string `json:"task_id"`
}

// TaskProgress 任务进度
type TaskProgress struct {
	TaskID   string `json:"task_id"`
	Progress uint32 `json:"progress"`
	Message  string `json:"message,omitempty"`
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	ExitCode   int32  `json:"exit_code"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	StartedAt  int64  `json:"started_at,omitempty"`
	FinishedAt int64  `json:"finished_at,omitempty"`
}