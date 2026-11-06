// Package config 基于 viper 的全局配置中心。
// 各模块通过 config.SetDefault(module, key, value) 注册默认值，
// 配置文件（默认 master.yaml）与环境变量（前缀 GW_）可覆盖。
package config

import (
	"github.com/spf13/viper"

	"gateway-manager/pkg/boot"
)

func init() {
	boot.Register("config", &boot.Task{
		Startup:  Startup,
		Shutdown: Shutdown,
	})

	// 环境变量：前缀 GW_，自动绑定
	viper.SetEnvPrefix("gw")
	viper.AutomaticEnv()

	// 通用默认值
	SetDefault("jwt", "secret", "gateway-manager-secret-key-change-me")
	SetDefault("system", "name", "Gateway RMM")
	SetDefault("auth", "admin_user", "admin")
	SetDefault("auth", "admin_password", "admin123")
	SetDefault("auth", "allow_auto_register", true)
}

// Name 设置配置文件名称（不含扩展名）
func Name(name string) { viper.SetConfigName(name) }

// AddPath 追加一个配置搜索目录
func AddPath(path string) { viper.AddConfigPath(path) }

// SetType 设置配置文件类型（默认 yaml）
func SetType(t string) { viper.SetConfigType(t) }

// Load 加载配置文件。auto=true 时文件不存在会自动生成默认配置。
func Load(auto bool) error {
	if err := viper.ReadInConfig(); err != nil {
		if auto {
			return viper.SafeWriteConfig()
		}
		return err
	}
	return nil
}

// SetDefault 注册模块默认值（module.key）
func SetDefault(module, key string, value any) {
	viper.SetDefault(module+"."+key, value)
}

// Set 运行时覆盖配置
func Set(module, key string, value any) { viper.Set(module+"."+key, value) }

// GetString 读字符串
func GetString(module, key string) string { return viper.GetString(module + "." + key) }

// GetBool 读布尔
func GetBool(module, key string) bool { return viper.GetBool(module + "." + key) }

// GetInt 读整数
func GetInt(module, key string) int { return viper.GetInt(module + "." + key) }

// GetFloat 读浮点
func GetFloat(module, key string) float64 { return viper.GetFloat64(module + "." + key) }

// GetStringSlice 读字符串数组
func GetStringSlice(module, key string) []string { return viper.GetStringSlice(module + "." + key) }

// Startup boot 启动入口：加载 master.yaml（auto=true 自动生成默认配置）
func Startup() error {
	Name("master")
	AddPath(".")
	return Load(true)
}

// Shutdown boot 关闭入口
func Shutdown() error { return nil }