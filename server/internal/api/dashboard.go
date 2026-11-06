package api

import (
	"github.com/gin-gonic/gin"

	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/db"
	"gateway-manager/server/internal/device"
)

// DashboardSummary 首页统计：设备总数 / 在线 / 离线
func DashboardSummary(c *gin.Context) {
	total, err := db.Count("device", nil)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	online := int64(device.Default.OnlineCount())
	offline := total - online
	if offline < 0 {
		offline = 0
	}
	curd.Success(c, gin.H{
		"total":   total,
		"online":  online,
		"offline": offline,
	})
}