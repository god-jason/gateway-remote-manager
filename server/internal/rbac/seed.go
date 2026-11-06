package rbac

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"xorm.io/builder"

	"gateway-manager/pkg/config"
	"gateway-manager/pkg/db"
	"gateway-manager/pkg/logger"
	"gateway-manager/server/internal/types"
)

// Seed 初始化默认角色与初始管理员用户（幂等，仅在缺失时创建）
func Seed() error {
	if err := seedRole("admin", "管理员（全部权限）", AllPermissions); err != nil {
		return err
	}
	if err := seedRole("viewer", "只读访客", ViewerPermissions); err != nil {
		return err
	}

	username := config.GetString("auth", "admin_user")
	if username == "" {
		username = "admin"
	}
	password := config.GetString("auth", "admin_password")
	if password == "" {
		password = "admin123"
	}

	var exist types.User
	if found, _ := db.FindOne("user", builder.Eq{"username": username}, &exist); found {
		return nil
	}

	var adminRole types.Role
	if found, _ := db.FindOne("role", builder.Eq{"name": "admin"}, &adminRole); !found {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &types.User{
		ID:       db.NewID(),
		Username: username,
		Password: string(hash),
		Nickname: "管理员",
		RoleID:   adminRole.ID,
		Status:   "active",
		Created:  time.Now(),
	}
	if _, err := db.InsertOne("user", user); err != nil {
		return err
	}
	logger.Info("已创建初始管理员", "username", username)
	return nil
}

func seedRole(name, desc string, perms []string) error {
	var r types.Role
	if found, _ := db.FindOne("role", builder.Eq{"name": name}, &r); found {
		return nil
	}
	role := &types.Role{
		ID:          db.NewID(),
		Name:        name,
		Description: desc,
		Permissions: Join(perms),
		Created:     time.Now(),
	}
	_, err := db.InsertOne("role", role)
	return err
}