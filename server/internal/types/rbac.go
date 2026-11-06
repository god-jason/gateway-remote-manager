package types

import "time"

// User 用户
type User struct {
	ID       string    `xorm:"'id' pk varchar(24)" json:"id"`
	Username string    `xorm:"username varchar(64) unique" json:"username"`
	Password string    `xorm:"password varchar(128)" json:"-"` // bcrypt 哈希
	Nickname string    `xorm:"nickname varchar(64)" json:"nickname"`
	Email    string    `xorm:"email varchar(128)" json:"email"`
	RoleID   string    `xorm:"role_id varchar(24)" json:"role_id"`
	Status   string    `xorm:"status varchar(16)" json:"status"` // active / disabled
	Created  time.Time `xorm:"created" json:"created"`
}

// Role 角色
type Role struct {
	ID          string    `xorm:"'id' pk varchar(24)" json:"id"`
	Name        string    `xorm:"name varchar(64) unique" json:"name"`
	Description string    `xorm:"description varchar(256)" json:"description"`
	Permissions string    `xorm:"permissions text" json:"permissions"` // 逗号分隔权限码
	Created     time.Time `xorm:"created" json:"created"`
}