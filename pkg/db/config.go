package db

import (
	"fmt"
	"strings"

	"gateway-manager/pkg/config"
)

// MODULE viper 模块名
const MODULE = "db"

func init() {
	config.SetDefault(MODULE, "enable", true)
	config.SetDefault(MODULE, "driver", "mysql")
	config.SetDefault(MODULE, "host", "127.0.0.1")
	config.SetDefault(MODULE, "port", 3306)
	config.SetDefault(MODULE, "database", "gateway_manager")
	config.SetDefault(MODULE, "username", "root")
	config.SetDefault(MODULE, "password", "")
	config.SetDefault(MODULE, "sslmode", "disable")
	config.SetDefault(MODULE, "params", "")
	config.SetDefault(MODULE, "debug", false)
	config.SetDefault(MODULE, "sync", true)
}

// Enabled 是否启用 DB（默认 true）
func Enabled() bool { return config.GetBool(MODULE, "enable") }

// DSN 由分散字段拼装连接串
func DSN(driver, host string, port int, database, username, password, sslmode, params string) (string, error) {
	switch strings.ToLower(driver) {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password, host, port, database)
		if params != "" {
			dsn += "?" + strings.TrimPrefix(params, "?")
		} else {
			dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
		}
		return dsn, nil
	case "postgres", "postgresql":
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", username, password, host, port, database, sslmode)
		if params != "" {
			dsn += "&" + strings.TrimPrefix(params, "&")
		}
		return dsn, nil
	default:
		return "", fmt.Errorf("unsupported driver: %s (valid: mysql, postgres)", driver)
	}
}