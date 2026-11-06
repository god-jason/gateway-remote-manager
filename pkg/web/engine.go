package web

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"gateway-manager/pkg/config"
)

// Engine Gin 引擎封装
type Engine struct {
	*gin.Engine
}

// New 创建 Web 引擎（含默认中间件与静态前端托管）
func New() *Engine {
	if config.GetBool(MODULE, "debug") {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(Recovery())
	engine.Use(CORS())

	e := &Engine{Engine: engine}

	if staticDir := config.GetString(MODULE, "static"); staticDir != "" {
		if err := e.ServeFrontend(staticDir); err != nil {
			if !os.IsNotExist(err) {
				panic("服务前端静态页面失败: " + err.Error())
			}
			// 目录不存在时静默跳过（开发环境前端独立启动）
		}
	}
	return e
}

// ServeFrontend 托管 SPA 静态资源
func (e *Engine) ServeFrontend(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absDir); err != nil {
		return err
	}

	e.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"code": 404, "message": "not found"})
			return
		}
		filePath := filepath.Join(absDir, path)
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath)
			return
		}
		c.File(filepath.Join(absDir, "index.html"))
	})
	return nil
}