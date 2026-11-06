// Package main Gateway Server 可执行程序入口。
// 启动流程：注册数据模型 → boot 拓扑启动（config/logger/db）→ Web 引擎 + API + Agent WS。
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway-manager/pkg/boot"
	"gateway-manager/pkg/config"
	"gateway-manager/pkg/db"
	"gateway-manager/pkg/logger"
	"gateway-manager/pkg/web"
	"gateway-manager/server/internal/api"
	"gateway-manager/server/internal/connection"
	"gateway-manager/server/internal/terminal"
	"gateway-manager/server/internal/types"
)

func main() {
	// 服务端模块默认值
	config.SetDefault("server", "heartbeat_interval", 30)
	config.SetDefault("server", "heartbeat_timeout", 90)
	config.SetDefault("upgrade", "dir", "./uploads/upgrades")
	config.SetDefault("upgrade", "base_url", "")

	// 数据模型注册（db.sync=true 时自动同步表结构）
	db.Register(new(types.Device), new(types.DeviceCredential), new(types.Config), new(types.Task), new(types.AgentRelease), new(types.UpgradeTask))

	if err := boot.Startup(); err != nil {
		logger.Error("启动失败", "error", err)
		_ = boot.Shutdown()
		os.Exit(1)
	}

	engine := web.New()
	api.InitRouter(engine.Group("/api"))
	connection.Register(engine.Engine)
	terminal.Register(engine.Engine)

	addr := fmt.Sprintf("%s:%d", config.GetString("web", "host"), config.GetInt("web", "port"))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("收到退出信号", "signal", sig.String())
		_ = boot.Shutdown()
		time.AfterFunc(3*time.Second, func() { os.Exit(0) })
	}()

	logger.Info("Gateway Server 启动", "address", addr)
	if err := engine.Run(addr); err != nil {
		logger.Error("Web 服务错误", "error", err)
	}
}