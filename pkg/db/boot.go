package db

import (
	"fmt"

	"gateway-manager/pkg/boot"
	"gateway-manager/pkg/config"
	"gateway-manager/pkg/logger"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"xorm.io/xorm"
	"xorm.io/xorm/log"
	"xorm.io/xorm/names"
)

func init() {
	boot.Register(MODULE, &boot.Task{
		Startup:  Startup,
		Shutdown: Shutdown,
		Depends:  []string{"config"},
	})
}

var Engine *xorm.Engine

// Startup boot 启动入口：按配置连接数据库并（可选）同步表结构
func Startup() error {
	if !Enabled() {
		logger.Info("db 模块已禁用，跳过启动")
		return nil
	}

	driver := config.GetString(MODULE, "driver")
	dsn, err := DSN(
		driver,
		config.GetString(MODULE, "host"),
		config.GetInt(MODULE, "port"),
		config.GetString(MODULE, "database"),
		config.GetString(MODULE, "username"),
		config.GetString(MODULE, "password"),
		config.GetString(MODULE, "sslmode"),
		config.GetString(MODULE, "params"),
	)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	engine, err := xorm.NewEngine(driver, dsn)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	engine.SetMapper(names.GonicMapper{})

	if config.GetBool(MODULE, "debug") {
		engine.ShowSQL(true)
		engine.SetLogLevel(log.LOG_DEBUG)
	}

	if config.GetBool(MODULE, "sync") {
		if err := engine.Sync2(models...); err != nil {
			logger.Warn("自动同步表结构失败", "error", err)
		}
	}

	Engine = engine
	logger.Info("数据库连接成功", "driver", driver)
	return nil
}

// Shutdown boot 关闭入口
func Shutdown() error {
	if Engine != nil {
		Engine.Close()
		Engine = nil
	}
	return nil
}