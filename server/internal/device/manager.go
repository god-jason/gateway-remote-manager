package device

import (
	"sync"
	"time"

	"gateway-manager/pkg/grpv1"
)

// LiveStatus 设备实时状态（内存维护，与数据库持久化状态分离）
type LiveStatus struct {
	Online        bool      `json:"online"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryTotal   uint64    `json:"memory_total"`
	MemoryUsed    uint64    `json:"memory_used"`
	DiskTotal     uint64    `json:"disk_total"`
	DiskUsed      uint64    `json:"disk_used"`
	NetworkRxByte uint64    `json:"network_rx_bytes"`
	NetworkTxByte uint64    `json:"network_tx_bytes"`
	Uptime        uint64    `json:"uptime"`
	Hostname      string    `json:"hostname"`
}

type entry struct {
	conn grpv1.Connection
	live LiveStatus
}

// Manager 连接与实时状态管理
type Manager struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

// NewManager 创建管理器
func NewManager() *Manager {
	return &Manager{entries: make(map[string]*entry)}
}

// Default 全局单例
var Default = NewManager()

// Register 注册连接（上线）
func (m *Manager) Register(conn grpv1.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[conn.DeviceID()]
	if e == nil {
		e = &entry{}
		m.entries[conn.DeviceID()] = e
	}
	e.conn = conn
	e.live.Online = true
	e.live.LastSeenAt = time.Now()
}

// Unregister 注销连接（离线）
func (m *Manager) Unregister(conn grpv1.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.entries[conn.DeviceID()]; e != nil && e.conn == conn {
		e.conn = nil
		e.live.Online = false
		e.live.LastSeenAt = time.Now()
	}
}

// Touch 更新最后活跃时间
func (m *Manager) Touch(deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.entries[deviceID]; e != nil {
		e.live.LastSeenAt = time.Now()
	}
}

// UpdateSystem 更新系统资源状态
func (m *Manager) UpdateSystem(deviceID string, st *grpv1.SystemStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[deviceID]
	if e == nil {
		e = &entry{}
		m.entries[deviceID] = e
	}
	e.live.CPUUsage = st.CPUUsage
	e.live.MemoryTotal = st.MemoryTotal
	e.live.MemoryUsed = st.MemoryUsed
	e.live.DiskTotal = st.DiskTotal
	e.live.DiskUsed = st.DiskUsed
	e.live.NetworkRxByte = st.NetworkRxByte
	e.live.NetworkTxByte = st.NetworkTxByte
	e.live.Uptime = st.Uptime
	e.live.Hostname = st.Hostname
	e.live.LastSeenAt = time.Now()
}

// Live 获取实时状态（在线才返回 true）
func (m *Manager) Live(deviceID string) (LiveStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e := m.entries[deviceID]; e != nil && e.conn != nil {
		return e.live, true
	}
	return LiveStatus{}, false
}

// Connection 获取设备连接
func (m *Manager) Connection(deviceID string) (grpv1.Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e := m.entries[deviceID]; e != nil && e.conn != nil {
		return e.conn, true
	}
	return nil, false
}

// OnlineCount 在线设备数
func (m *Manager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, e := range m.entries {
		if e.conn != nil {
			n++
		}
	}
	return n
}