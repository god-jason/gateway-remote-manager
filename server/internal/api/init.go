// Package api Web 管理端 HTTP API。
package api

import "github.com/gin-gonic/gin"

// InitRouter 注册 /api 路由：先公开登录，其余走 JWT 校验
func InitRouter(r *gin.RouterGroup) {
	r.POST("/login", Login)

	r.Use(JWTAuth())

	r.GET("/me", Me)
	r.GET("/dashboard/summary", DashboardSummary)

	r.POST("/device/search", DeviceSearch)
	r.GET("/device/:id", DeviceGet)
	r.DELETE("/device/:id", DeviceDelete)
}