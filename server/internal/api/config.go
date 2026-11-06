package api

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xorm.io/builder"

	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/db"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/device"
	"gateway-manager/server/internal/types"
)

// registerConfigRoutes 注册配置管理路由（JWT 之后）
func registerConfigRoutes(r *gin.RouterGroup) {
	r.POST("/device/:deviceId/config/list", ConfigList)
	r.GET("/device/:deviceId/config/get", ConfigGet)
	r.POST("/device/:deviceId/config/set", ConfigSet)
	r.POST("/device/:deviceId/config/delete", ConfigDelete)
}

// configView 配置视图（期望状态 + 与上报状态的同步标记）
type configView struct {
	ConfigKey  string    `json:"config_key"`
	ConfigType string    `json:"config_type"`
	Version    uint64    `json:"version"`
	Checksum   string    `json:"checksum"`
	Status     string    `json:"status"`
	Synced     bool      `json:"synced"`
	Updated    time.Time `json:"updated"`
}

// ConfigList 配置列表（期望状态，叠加与 Agent 上报状态的一致性）
func ConfigList(c *gin.Context) {
	deviceID := c.Param("deviceId")

	var cfgs []types.Config
	if err := db.Find("config", builder.Eq{"device_id": deviceID}, &cfgs, 0, 0); err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}

	reported := map[string]string{}
	if conn, ok := device.Default.Connection(deviceID); ok {
		env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeConfigList, deviceID)
		env.Payload, _ = grpv1.MarshalPayload(struct{}{})
		if resp, err := conn.Request(c.Request.Context(), env, 10*time.Second); err == nil {
			var list grpv1.ConfigListResult
			if grpv1.UnmarshalPayload(resp.Payload, &list) == nil && list.Success {
				for _, c := range list.Configs {
					reported[c.ConfigKey] = c.Checksum
				}
			}
		}
	}

	views := make([]configView, 0, len(cfgs))
	for _, c := range cfgs {
		views = append(views, configView{
			ConfigKey:  c.ConfigKey,
			ConfigType: c.ConfigType,
			Version:    c.Version,
			Checksum:   c.Checksum,
			Status:     c.Status,
			Updated:    c.Updated,
			Synced:     c.Checksum != "" && reported[c.ConfigKey] == c.Checksum,
		})
	}
	curd.Success(c, views)
}

// ConfigGet 获取期望配置详情
func ConfigGet(c *gin.Context) {
	deviceID := c.Param("deviceId")
	configKey := c.Query("config_key")

	var cfg types.Config
	found, err := db.FindOne("config", builder.And(
		builder.Eq{"device_id": deviceID},
		builder.Eq{"config_key": configKey},
	), &cfg)
	if err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}
	if !found {
		curd.ErrorWithStatus(c, 404, 404, "配置不存在")
		return
	}
	curd.Success(c, cfg)
}

// ConfigSet 保存期望配置并下发到 Agent
func ConfigSet(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req struct {
		ConfigKey  string `json:"config_key"`
		ConfigType string `json:"config_type"`
		Content    string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	if req.ConfigKey == "" {
		curd.Error(c, 400, "缺少 config_key")
		return
	}
	if req.ConfigType == "" {
		req.ConfigType = typeFromKey(req.ConfigKey)
	}
	sum := checksumOf(req.Content)
	now := time.Now()

	cond := builder.And(builder.Eq{"device_id": deviceID}, builder.Eq{"config_key": req.ConfigKey})
	var cfg types.Config
	found, err := db.FindOne("config", cond, &cfg)
	if err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}

	version := uint64(1)
	if found {
		version = cfg.Version + 1
		_, _ = db.UpdateOne("config", cond, map[string]interface{}{
			"content":     req.Content,
			"config_type": req.ConfigType,
			"checksum":    sum,
			"version":     version,
			"status":      "pending",
			"updated":     now,
		})
	} else {
		cfg = types.Config{
			ID:         db.NewID(),
			DeviceID:   deviceID,
			ConfigKey:  req.ConfigKey,
			ConfigType: req.ConfigType,
			Content:    req.Content,
			Checksum:   sum,
			Version:    version,
			Status:     "pending",
			Updated:    now,
			Created:    now,
		}
		if _, err := db.InsertOne("config", &cfg); err != nil {
			curd.ErrorWithStatus(c, 500, 500, err.Error())
			return
		}
	}

	// 下发到 Agent
	status := "pending"
	synced := false
	if conn, ok := device.Default.Connection(deviceID); ok {
		env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeConfigSet, deviceID)
		env.Payload, _ = grpv1.MarshalPayload(grpv1.ConfigSetRequest{ConfigKey: req.ConfigKey, Content: req.Content, Checksum: sum})
		resp, err := conn.Request(c.Request.Context(), env, 15*time.Second)
		if err != nil {
			status = "pending"
		} else {
			var r grpv1.ConfigResult
			if grpv1.UnmarshalPayload(resp.Payload, &r) == nil && r.Success && r.Config != nil && r.Config.Checksum == sum {
				status = "applied"
				synced = true
			} else {
				status = "failed"
			}
		}
		_, _ = db.UpdateOne("config", cond, map[string]interface{}{"status": status})
	}

	curd.Success(c, configView{
		ConfigKey:  req.ConfigKey,
		ConfigType: req.ConfigType,
		Version:    version,
		Checksum:   sum,
		Status:     status,
		Synced:     synced,
		Updated:    now,
	})
}

// ConfigDelete 删除期望配置，并删除 Agent 侧配置
func ConfigDelete(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req grpv1.ConfigKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	cond := builder.And(builder.Eq{"device_id": deviceID}, builder.Eq{"config_key": req.ConfigKey})
	if _, err := db.DeleteOne("config", cond); err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return
	}
	if conn, ok := device.Default.Connection(deviceID); ok {
		env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeConfigDelete, deviceID)
		env.Payload, _ = grpv1.MarshalPayload(req)
		_, _ = conn.Request(c.Request.Context(), env, 10*time.Second)
	}
	curd.Success(c, nil)
}

func checksumOf(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func typeFromKey(key string) string {
	switch {
	case strings.HasSuffix(key, ".yaml") || strings.HasSuffix(key, ".yml"):
		return "yaml"
	case strings.HasSuffix(key, ".json"):
		return "json"
	case strings.HasSuffix(key, ".toml"):
		return "toml"
	case strings.HasSuffix(key, ".ini"):
		return "ini"
	default:
		return "text"
	}
}