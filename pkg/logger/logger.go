// Package logger 基于标准库 log/slog 的统一日志。
package logger

import (
	"log/slog"
	"os"

	"gateway-manager/pkg/boot"
	"gateway-manager/pkg/config"
)

const MODULE = "log"

var defaultLogger *slog.Logger

func init() {
	config.SetDefault(MODULE, "level", "info")

	defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(defaultLogger)

	boot.Register(MODULE, &boot.Task{
		Startup: func() error {
			Init(config.GetString(MODULE, "level"))
			return nil
		},
		Shutdown: func() error { return nil },
		Depends:  []string{"config"},
	})
}

// Init 按级别初始化日志
func Init(level string) {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
	slog.SetDefault(defaultLogger)
}

func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func Info(msg string, args ...any)  { slog.Info(msg, args...) }
func Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }