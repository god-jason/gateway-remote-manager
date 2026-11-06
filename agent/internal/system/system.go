// Package system 采集网关运行状态（CPU/内存/磁盘/网络/运行时长）。
package system

import (
	"os"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"

	"gateway-manager/pkg/grpv1"
)

// Collect 采集系统状态，采集失败时对应字段置零，不中断整体流程
func Collect(agentUptime time.Duration) grpv1.SystemStatus {
	st := grpv1.SystemStatus{}

	if h, err := os.Hostname(); err == nil {
		st.Hostname = h
	}
	if u, err := host.Uptime(); err == nil {
		st.Uptime = u
	}
	if v, err := mem.VirtualMemory(); err == nil && v != nil {
		st.MemoryTotal = v.Total
		st.MemoryUsed = v.Used
	}
	if d, err := disk.Usage("/"); err == nil && d != nil {
		st.DiskTotal = d.Total
		st.DiskUsed = d.Used
	}
	if p, err := cpu.Percent(0, false); err == nil && len(p) > 0 {
		st.CPUUsage = round1(p[0])
	}
	if ios, err := net.IOCounters(false); err == nil && len(ios) > 0 {
		st.NetworkRxByte = ios[0].BytesRecv
		st.NetworkTxByte = ios[0].BytesSent
	}
	return st
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}