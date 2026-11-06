// Package terminal Agent 端远程终端：PTY 与 Stream 之间的双向转发管理。
package terminal

import (
	"context"
	"sync"

	"gateway-manager/pkg/grpv1"
	"gateway-manager/pkg/logger"
)

type session struct {
	id     string
	term   *Terminal
	stream grpv1.Stream
	once   sync.Once
}

// Manager 终端会话管理器
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
}

// NewManager 创建管理器
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*session)}
}

// Open 建立终端会话，并在 PTY 与 Stream 之间双向转发
func (m *Manager) Open(ctx context.Context, open grpv1.TerminalOpen, stream grpv1.Stream) error {
	term, err := Start(open.Shell, uint16(open.Rows), uint16(open.Cols), open.WorkingDirectory)
	if err != nil {
		return err
	}
	s := &session{id: open.SessionID, term: term, stream: stream}

	m.mu.Lock()
	m.sessions[open.SessionID] = s
	m.mu.Unlock()

	go m.pumpOutput(s) // PTY 输出 -> Stream
	go m.pumpInput(s)  // Stream -> PTY 输入

	logger.Info("终端会话已建立", "session", open.SessionID, "shell", open.Shell)
	return nil
}

// Resize 调整终端窗口
func (m *Manager) Resize(sessionID string, rows, cols uint16) error {
	m.mu.Lock()
	s := m.sessions[sessionID]
	m.mu.Unlock()
	if s == nil {
		return grpv1.NewError(grpv1.ErrSessionNotFound, "会话不存在")
	}
	return s.term.Resize(rows, cols)
}

// Close 关闭指定会话
func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	s := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	if s != nil {
		s.term.Close()
		_ = s.stream.Close()
	}
}

// CloseAll 关闭全部会话（连接断开时清理）
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := make([]*session, 0, len(m.sessions))
	for id, s := range m.sessions {
		sessions = append(sessions, s)
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	for _, s := range sessions {
		s.term.Close()
		_ = s.stream.Close()
	}
}

// pumpOutput 读取 PTY 输出写入 Stream（发往浏览器）
func (m *Manager) pumpOutput(s *session) {
	buf := make([]byte, 4096)
	for {
		n, err := s.term.File.Read(buf)
		if n > 0 {
			if _, werr := s.stream.Write(context.Background(), buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	m.teardown(s)
}

// pumpInput 读取 Stream 输入写入 PTY（来自浏览器）
func (m *Manager) pumpInput(s *session) {
	buf := make([]byte, 4096)
	for {
		n, err := s.stream.Read(context.Background(), buf)
		if n > 0 {
			if _, werr := s.term.File.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	// 远端关闭流，终止 PTY
	s.term.Close()
	m.teardown(s)
}

func (m *Manager) teardown(s *session) {
	s.once.Do(func() {
		m.mu.Lock()
		delete(m.sessions, s.id)
		m.mu.Unlock()
		s.term.Close()
		_ = s.stream.Close()
		logger.Info("终端会话已关闭", "session", s.id)
	})
}