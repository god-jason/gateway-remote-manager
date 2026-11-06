// Package task Agent 端任务执行：shell 命令执行、超时、取消与结果回报。
package task

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"gateway-manager/pkg/grpv1"
)

const maxOutputBytes = 1024 * 1024

// OnResult 结果回调
type OnResult func(grpv1.TaskResult)

// Manager 任务管理器
type Manager struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// NewManager 创建管理器
func NewManager() *Manager {
	return &Manager{cancels: make(map[string]context.CancelFunc)}
}

// Cancel 取消任务
func (m *Manager) Cancel(taskID string) error {
	m.mu.Lock()
	cancel := m.cancels[taskID]
	m.mu.Unlock()
	if cancel == nil {
		return grpv1.NewError(grpv1.ErrTaskNotFound, "任务不存在或已结束")
	}
	cancel()
	return nil
}

// Execute 启动任务（异步执行，立即返回）
func (m *Manager) Execute(ctx context.Context, create grpv1.TaskCreate, on OnResult) error {
	m.mu.Lock()
	if _, exists := m.cancels[create.TaskID]; exists {
		m.mu.Unlock()
		return grpv1.NewError(grpv1.ErrTaskConflict, "任务已存在")
	}
	taskCtx, cancel := context.WithCancel(context.Background())
	if create.TimeoutSeconds > 0 {
		taskCtx, cancel = context.WithTimeout(taskCtx, time.Duration(create.TimeoutSeconds)*time.Second)
	}
	m.cancels[create.TaskID] = cancel
	m.mu.Unlock()

	go m.run(taskCtx, cancel, create, on)
	return nil
}

func (m *Manager) run(ctx context.Context, cancel context.CancelFunc, create grpv1.TaskCreate, on OnResult) {
	defer func() {
		cancel()
		m.mu.Lock()
		delete(m.cancels, create.TaskID)
		m.mu.Unlock()
	}()

	started := time.Now()
	var buf limitedBuffer
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", create.Command)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Env = os.Environ()

	err := cmd.Run()

	status := grpv1.TaskStatusSuccess
	exitCode := int32(0)
	var exitErr *exec.ExitError
	if err != nil {
		switch {
		case ctx.Err() == context.DeadlineExceeded:
			status = grpv1.TaskStatusTimeout
		case ctx.Err() == context.Canceled:
			status = grpv1.TaskStatusCancelled
		case errors.As(err, &exitErr):
			status = grpv1.TaskStatusFailed
			exitCode = int32(exitErr.ExitCode())
		default:
			status = grpv1.TaskStatusFailed
		}
	}

	res := grpv1.TaskResult{
		TaskID:     create.TaskID,
		Status:     status,
		ExitCode:   exitCode,
		Output:     buf.String(),
		StartedAt:  started.UnixMilli(),
		FinishedAt: time.Now().UnixMilli(),
	}
	if err != nil && status == grpv1.TaskStatusFailed && exitCode == 0 {
		res.Error = err.Error()
	}
	on(res)
}

// limitedBuffer 限制输出大小，防止内存无限增长
type limitedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	orig := len(p)
	remaining := maxOutputBytes - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return orig, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buf.Write(p)
	return orig, nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.truncated {
		return b.buf.String() + "\n...(输出已截断)"
	}
	return b.buf.String()
}