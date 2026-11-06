package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"xorm.io/builder"

	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/db"
	"gateway-manager/server/internal/device"
	"gateway-manager/server/internal/types"
)

// DeviceView 设备视图：持久化字段 + 实时状态
type DeviceView struct {
	types.Device
	Online        bool      `json:"online"`
	SeenAt        time.Time `json:"seen_at,omitempty"`
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

// DeviceSearch 分页搜索设备（叠加在线状态与资源使用）
func DeviceSearch(c *gin.Context) {
	var params curd.ListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		curd.Error(c, 400, "无效的请求参数")
		return
	}
	params.Normalize()

	cond := params.BuildCond()
	if params.HasKeyword() {
		kw := builder.Or(
			builder.Like{"name", "%" + params.Keyword + "%"},
			builder.Like{"device_id", "%" + params.Keyword + "%"},
		)
		if cond != nil {
			cond = builder.And(cond, kw)
		} else {
			cond = kw
		}
	}

	total, err := db.Count("device", cond)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	list := make([]types.Device, 0, params.PageSize)
	if err := db.PagedFind("device", cond, params.Page, params.PageSize, params.OrderBy, params.Order == "asc", &list); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}

	items := make([]DeviceView, 0, len(list))
	for _, d := range list {
		v := DeviceView{Device: d, SeenAt: d.LastSeenAt}
		if live, ok := device.Default.Live(d.DeviceID); ok {
			v.Online = true
			v.SeenAt = live.LastSeenAt
			v.CPUUsage = live.CPUUsage
			v.MemoryTotal = live.MemoryTotal
			v.MemoryUsed = live.MemoryUsed
			v.DiskTotal = live.DiskTotal
			v.DiskUsed = live.DiskUsed
			v.NetworkRxByte = live.NetworkRxByte
			v.NetworkTxByte = live.NetworkTxByte
			v.Uptime = live.Uptime
			v.Hostname = live.Hostname
		}
		items = append(items, v)
	}
	curd.SuccessTotal(c, items, total)
}

// DeviceGet 设备详情（叠加实时状态）
func DeviceGet(c *gin.Context) {
	var d types.Device
	found, err := db.FindOne("device", builder.Eq{"id": c.Param("id")}, &d)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	if !found {
		curd.Error(c, 404, "数据不存在")
		return
	}
	v := DeviceView{Device: d, SeenAt: d.LastSeenAt}
	if live, ok := device.Default.Live(d.DeviceID); ok {
		v.Online = true
		v.SeenAt = live.LastSeenAt
		v.CPUUsage = live.CPUUsage
		v.MemoryTotal = live.MemoryTotal
		v.MemoryUsed = live.MemoryUsed
		v.DiskTotal = live.DiskTotal
		v.DiskUsed = live.DiskUsed
		v.Hostname = live.Hostname
	}
	curd.Success(c, v)
}

// DeviceDelete 删除设备（及其凭据）
func DeviceDelete(c *gin.Context) {
	var d types.Device
	found, err := db.FindOne("device", builder.Eq{"id": c.Param("id")}, &d)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	if !found {
		curd.Error(c, 404, "数据不存在")
		return
	}
	if _, err := db.DeleteOne("device", builder.Eq{"id": d.ID}); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	_, _ = db.DeleteOne("device_credential", builder.Eq{"device_id": d.DeviceID})
	curd.Success(c, nil)
}