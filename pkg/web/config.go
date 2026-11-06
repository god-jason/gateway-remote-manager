package web

import "gateway-manager/pkg/config"

// MODULE viper 模块名
const MODULE = "web"

func init() {
	config.SetDefault(MODULE, "host", "0.0.0.0")
	config.SetDefault(MODULE, "port", 8080)
	config.SetDefault(MODULE, "static", "web/dist")
	config.SetDefault(MODULE, "debug", false)
}