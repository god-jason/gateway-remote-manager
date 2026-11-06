package api

import (
	"context"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"xorm.io/builder"

	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/db"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/device"
	"gateway-manager/server/internal/types"
)

// registerTaskRoutes 注册任务管理路由（JWT 之后）
func registerTaskRoutes(r *gin.RouterGroup) {
	r.POST("/device/:deviceId/task/create", TaskCreate)
	r.POST("/device/:deviceId/task/list", TaskList)
	r.GET("/device/:deviceId/task/get", TaskGet)
	r.POST("/device/:deviceId/task/cancel", TaskCancel)
}

func taskCond(deviceID, taskID string) builder.Cond {
	return builder.And(builder.Eq{"device_id": deviceID}, builder.Eq{"task_id": taskID})
}

// TaskCreate 创建并下发命令任务
func TaskCreate(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req struct {
		TaskType       string `json:"task_type"`
		Command        string `json:"command"`
		TimeoutSeconds uint32 `json:"timeout_seconds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	if req.Command == "" {
		curd.Error(c, 400, "缺少 command")
		return
	}
	if req.TaskType == "" {
		req.TaskType = grpv1.TaskTypeCommand
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 60
	}

	task := types.Task{
		ID:       db.NewID(),
		TaskID:   uuid.NewString(),
		DeviceID: deviceID,
		TaskType: req.TaskType,
		Command:  req.Command,
		Timeout:  req.TimeoutSeconds,
		Status:   "pending",
		Created:  time.Now(),
	}
	if _, err := db.InsertOne("task", &task); err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}

	if conn, ok := device.Default.Connection(deviceID); ok {
		dispatchTask(conn, &task)
	}

	curd.Success(c, task)
}

// dispatchTask 下发任务到 Agent 并订阅结果
func dispatchTask(conn grpv1.Connection, task *types.Task) {
	env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeTaskCreate, task.DeviceID)
	env.Payload, _ = grpv1.MarshalPayload(grpv1.TaskCreate{
		TaskID: task.TaskID, TaskType: task.TaskType, Command: task.Command, TimeoutSeconds: task.Timeout,
	})

	resp, err := conn.Request(context.Background(), env, 5*time.Second)
	if err != nil {
		updateTaskStatus(task.DeviceID, task.TaskID, "error", err.Error(), nil)
		return
	}
	var ack grpv1.TaskCreateResult
	_ = grpv1.UnmarshalPayload(resp.Payload, &ack)
	if !ack.Success {
		updateTaskStatus(task.DeviceID, task.TaskID, "error", ack.Message, nil)
		return
	}
	updateTaskStatus(task.DeviceID, task.TaskID, "dispatched", "", nil)

	// 订阅结果事件（异步更新）
	_ = conn.SubscribeEvent(func(ev *grpv1.Envelope) {
		if ev.Type != grpv1.TypeTaskResult {
			return
		}
		var r grpv1.TaskResult
		if grpv1.UnmarshalPayload(ev.Payload, &r) != nil || r.TaskID != task.TaskID {
			return
		}
		updateTaskStatus(task.DeviceID, task.TaskID, r.Status, r.Error, map[string]interface{}{
			"exit_code":   r.ExitCode,
			"output":      r.Output,
			"error":       r.Error,
			"started_at":  unixMilli(r.StartedAt),
			"finished_at": unixMilli(r.FinishedAt),
		})
	})
}

// TaskList 任务列表（不含大输出）
func TaskList(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var tasks []types.Task
	if err := db.Find("task", builder.Eq{"device_id": deviceID}, &tasks, 0, 0); err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Created.After(tasks[j].Created) })
	views := make([]types.Task, 0, len(tasks))
	for _, t := range tasks {
		t.Output = ""
		views = append(views, t)
	}
	curd.Success(c, views)
}

// TaskGet 任务详情（含输出）
func TaskGet(c *gin.Context) {
	deviceID := c.Param("deviceId")
	taskID := c.Query("task_id")
	var task types.Task
	found, err := db.FindOne("task", taskCond(deviceID, taskID), &task)
	if err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}
	if !found {
		curd.ErrorWithStatus(c, 404, 404, "任务不存在")
		return
	}
	curd.Success(c, task)
}

// TaskCancel 取消任务
func TaskCancel(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req grpv1.TaskCancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var task types.Task
	found, _ := db.FindOne("task", taskCond(deviceID, req.TaskID), &task)
	if !found {
		curd.ErrorWithStatus(c, 404, 404, "任务不存在")
		return
	}
	if isTerminalStatus(task.Status) {
		curd.Success(c, nil)
		return
	}

	if conn, ok := device.Default.Connection(deviceID); ok {
		env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeTaskCancel, deviceID)
		env.Payload, _ = grpv1.MarshalPayload(req)
		_, _ = conn.Request(c.Request.Context(), env, 5*time.Second)
	}
	updateTaskStatus(deviceID, req.TaskID, "cancelled", "", nil)
	curd.Success(c, nil)
}

func updateTaskStatus(deviceID, taskID, status, errMsg string, extra map[string]interface{}) {
	fields := map[string]interface{}{"status": status}
	if errMsg != "" {
		fields["error"] = errMsg
	}
	for k, v := range extra {
		fields[k] = v
	}
	_, _ = db.UpdateOne("task", taskCond(deviceID, taskID), fields)
}

func isTerminalStatus(status string) bool {
	switch status {
	case "success", "failed", "timeout", "cancelled", "error":
		return true
	}
	return false
}

func unixMilli(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}