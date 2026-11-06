package types

import "time"

// Task 远程任务
type Task struct {
	ID         string    `xorm:"'id' pk varchar(24)" json:"id"`
	TaskID     string    `xorm:"task_id varchar(64)" json:"task_id"`
	DeviceID   string    `xorm:"device_id varchar(64)" json:"device_id"`
	TaskType   string    `xorm:"task_type varchar(32)" json:"task_type"`
	Command    string    `xorm:"command text" json:"command"`
	Timeout    uint32    `xorm:"timeout_seconds" json:"timeout_seconds"`
	Status     string    `xorm:"status varchar(32)" json:"status"` // pending/dispatched/success/failed/timeout/cancelled/error
	ExitCode   int32     `xorm:"exit_code" json:"exit_code"`
	Output     string    `xorm:"output text" json:"output"`
	Error      string    `xorm:"error varchar(512)" json:"error"`
	StartedAt  time.Time `xorm:"started_at" json:"started_at"`
	FinishedAt time.Time `xorm:"finished_at" json:"finished_at"`
	Created    time.Time `xorm:"created" json:"created"`
}