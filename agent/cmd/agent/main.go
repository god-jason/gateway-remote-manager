// Package main 网关 Agent 可执行程序入口。
// 启动后加载本地身份、连接服务器、完成握手认证，并周期性上报心跳与系统状态。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"gateway-manager/agent/internal/core"
	"gateway-manager/pkg/config"
	"gateway-manager/pkg/logger"
)

func main() {
	var (
		server       string
		deviceID     string
		deviceKey    string
		identityFile string
		heartbeat    int
		report       int
		verbose      bool
		help         bool
	)
	pflag.StringVar(&server, "server", "", "服务器地址，如 ws://127.0.0.1:8080/agent/ws")
	pflag.StringVar(&deviceID, "device-id", "", "设备 ID（可选，首次初始化身份时生效）")
	pflag.StringVar(&deviceKey, "device-key", "", "设备密钥（可选）")
	pflag.StringVar(&identityFile, "id-file", "", "身份文件路径")
	pflag.IntVar(&heartbeat, "heartbeat", 0, "心跳间隔（秒）")
	pflag.IntVar(&report, "report", 0, "状态上报间隔（秒）")
	pflag.BoolVar(&verbose, "verbose", false, "调试日志")
	pflag.BoolVar(&help, "help", false, "帮助")
	pflag.Parse()

	if help {
		pflag.Usage()
		return
	}

	// 加载配置（agent.yaml + 环境变量 GW_*）
	config.SetType("yaml")
	config.Name("agent")
	config.AddPath(".")
	config.AddPath("agent/configs")
	config.SetDefault("agent", "server", "ws://127.0.0.1:8080/agent/ws")
	config.SetDefault("agent", "identity_file", "agent_identity.json")
	config.SetDefault("agent", "heartbeat_interval", 30)
	config.SetDefault("agent", "report_interval", 30)
	config.SetDefault("agent", "fs_root", "/")
	config.SetDefault("agent", "config_root", "configs")
	config.SetDefault("log", "level", "info")
	if err := config.Load(true); err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 命令行覆盖
	if server != "" {
		config.Set("agent", "server", server)
	}
	if identityFile != "" {
		config.Set("agent", "identity_file", identityFile)
	}
	if heartbeat > 0 {
		config.Set("agent", "heartbeat_interval", heartbeat)
	}
	if report > 0 {
		config.Set("agent", "report_interval", report)
	}

	level := config.GetString("log", "level")
	if verbose {
		level = "debug"
	}
	logger.Init(level)

	cfg := core.Config{
		Server:            config.GetString("agent", "server"),
		HeartbeatInterval: time.Duration(config.GetInt("agent", "heartbeat_interval")) * time.Second,
		ReportInterval:    time.Duration(config.GetInt("agent", "report_interval")) * time.Second,
		IdentityFile:      config.GetString("agent", "identity_file"),
		DeviceID:          deviceID,
		DeviceKey:         deviceKey,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("Gateway Agent 启动", "server", cfg.Server, "version", core.AgentVersion)
	if err := core.New(cfg).Run(ctx); err != nil {
		logger.Error("Agent 退出", "error", err)
		os.Exit(1)
	}
	logger.Info("Agent 正常退出")
}