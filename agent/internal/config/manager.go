// Package config Agent 端配置管理：以受控根目录保存配置，提供备份、原子替换与校验。
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gateway-manager/pkg/grpv1"
)

// Manager 配置管理器
type Manager struct {
	root      string
	backupDir string
}

// NewManager 创建管理器
func NewManager(root string) *Manager {
	if root == "" {
		root = "configs"
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return &Manager{root: filepath.Clean(root), backupDir: filepath.Join(filepath.Clean(root), ".backups")}
}

// path 将 config_key 映射到受控根目录内的文件路径
func (m *Manager) path(key string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash("/" + strings.TrimPrefix(key, "/")))
	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("无效配置 key: %s", key)
	}
	abs := filepath.Join(m.root, cleaned)
	if rel, err := filepath.Rel(m.root, abs); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("配置 key 越界: %s", key)
	}
	return abs, nil
}

// List 列出所有配置（元信息，不含内容）
func (m *Manager) List() ([]grpv1.ConfigObject, error) {
	var out []grpv1.ConfigObject
	_ = filepath.Walk(m.root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasPrefix(p, m.backupDir) {
			return nil
		}
		rel, rerr := filepath.Rel(m.root, p)
		if rerr != nil {
			return nil
		}
		out = append(out, grpv1.ConfigObject{
			ConfigKey:  filepath.ToSlash(rel),
			ConfigType: typeFromExt(rel),
			ModifiedAt: info.ModTime().Unix(),
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ConfigKey < out[j].ConfigKey })
	return out, nil
}

// Get 读取配置（含内容与校验和）
func (m *Manager) Get(key string) (*grpv1.ConfigObject, bool, error) {
	p, err := m.path(key)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	info, _ := os.Stat(p)
	mt := int64(0)
	if info != nil {
		mt = info.ModTime().Unix()
	}
	return &grpv1.ConfigObject{
		ConfigKey:  key,
		ConfigType: typeFromExt(key),
		Content:    string(data),
		Checksum:   checksum(data),
		ModifiedAt: mt,
	}, true, nil
}

// Set 写入配置：校验、备份、原子替换
func (m *Manager) Set(key, content, expectChecksum string) (*grpv1.ConfigObject, error) {
	p, err := m.path(key)
	if err != nil {
		return nil, err
	}
	sum := checksum([]byte(content))
	if expectChecksum != "" && expectChecksum != sum {
		return nil, fmt.Errorf("checksum 不匹配")
	}
	// 已有配置先备份
	if _, err := os.Stat(p); err == nil {
		if err := m.backup(key); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, p); err != nil {
		return nil, err
	}
	return &grpv1.ConfigObject{
		ConfigKey:  key,
		ConfigType: typeFromExt(key),
		Content:    content,
		Checksum:   sum,
		ModifiedAt: time.Now().Unix(),
	}, nil
}

// Delete 删除配置（删除前备份）
func (m *Manager) Delete(key string) error {
	p, err := m.path(key)
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); err == nil {
		_ = m.backup(key)
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Rollback 从最近备份恢复
func (m *Manager) Rollback(key string) error {
	p, err := m.path(key)
	if err != nil {
		return err
	}
	matches, _ := filepath.Glob(filepath.Join(m.backupDir, filepath.Base(p)+".*.bak"))
	if len(matches) == 0 {
		return fmt.Errorf("没有可用的备份")
	}
	sort.Strings(matches)
	latest := matches[len(matches)-1]
	data, err := os.ReadFile(latest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// backup 备份配置到 .backups 目录（时间戳命名）
func (m *Manager) backup(key string) error {
	p, err := m.path(key)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return err
	}
	dst := filepath.Join(m.backupDir, filepath.Base(p)+"."+strconv.FormatInt(time.Now().UnixMilli(), 10)+".bak")
	return os.WriteFile(dst, data, 0644)
}

func checksum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func typeFromExt(key string) string {
	switch strings.ToLower(filepath.Ext(key)) {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".ini":
		return "ini"
	default:
		return "text"
	}
}