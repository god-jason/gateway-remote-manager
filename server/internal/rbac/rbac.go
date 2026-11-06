// Package rbac 定义权限码与角色权限的解析/校验能力。
package rbac

import "strings"

// AllPermissions 全部权限码（对应设计文档第 20 节）
var AllPermissions = []string{
	"device.read", "device.manage",
	"system.read", "system.manage",
	"terminal.read", "terminal.connect",
	"filesystem.read", "filesystem.write", "filesystem.delete",
	"config.read", "config.write", "config.rollback",
	"process.read", "process.manage",
	"service.read", "service.manage",
	"upgrade.read", "upgrade.execute",
	"task.create", "task.cancel",
	"audit.read",
	"user.manage", "role.manage",
}

// ViewerPermissions 只读角色权限
var ViewerPermissions = []string{
	"device.read", "system.read", "terminal.read",
	"filesystem.read", "config.read", "upgrade.read",
}

// 常用权限码常量
const (
	PermDeviceRead       = "device.read"
	PermDeviceManage     = "device.manage"
	PermTerminalConnect  = "terminal.connect"
	PermFilesystemRead   = "filesystem.read"
	PermFilesystemWrite  = "filesystem.write"
	PermFilesystemDelete = "filesystem.delete"
	PermConfigRead       = "config.read"
	PermConfigWrite      = "config.write"
	PermTaskCreate       = "task.create"
	PermTaskCancel       = "task.cancel"
	PermUpgradeRead      = "upgrade.read"
	PermUpgradeExecute   = "upgrade.execute"
	PermUserManage       = "user.manage"
	PermRoleManage       = "role.manage"
)

// Split 解析逗号分隔的权限串
func Split(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Join 拼接权限串
func Join(perms []string) string { return strings.Join(perms, ",") }

// Has 判断是否拥有权限（支持通配 "*"）
func Has(perms []string, perm string) bool {
	for _, p := range perms {
		if p == "*" || p == perm {
			return true
		}
	}
	return false
}