package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/builder"

	"gateway-manager/pkg/config"
	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/db"
	"gateway-manager/server/internal/rbac"
	"gateway-manager/server/internal/types"
)

// Login 账号密码登录，返回 JWT
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效的请求参数")
		return
	}

	var user types.User
	found, err := db.FindOne("user", builder.Eq{"username": req.Username}, &user)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	if !found || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		curd.Error(c, 401, "用户名或密码错误")
		return
	}
	if user.Status != "active" {
		curd.Error(c, 401, "账号已被禁用")
		return
	}

	token, err := issueToken(user.ID, user.Username)
	if err != nil {
		curd.Error(c, 500, "生成令牌失败")
		return
	}
	curd.Success(c, gin.H{"token": token, "username": user.Username, "nickname": user.Nickname})
}

// Me 当前登录用户（含角色与权限）
func Me(c *gin.Context) {
	curd.Success(c, gin.H{
		"username":    c.GetString("username"),
		"role_id":     c.GetString("role_id"),
		"permissions": c.GetStringSlice("permissions"),
	})
}

func issueToken(userID, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(config.GetString("jwt", "secret")))
}

// JWTAuth JWT 校验中间件：解析 token，加载用户与角色权限
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := AuthenticateToken(extractToken(c))
		if err != nil {
			curd.Unauthorized(c, "未登录或登录已过期")
			c.Abort()
			return
		}
		userID, _ := claims["sub"].(string)

		var user types.User
		found, _ := db.FindOne("user", builder.Eq{"id": userID}, &user)
		if !found || user.Status != "active" {
			curd.Unauthorized(c, "账号不存在或已禁用")
			c.Abort()
			return
		}

		perms := []string{}
		if user.RoleID != "" {
			var role types.Role
			if f, _ := db.FindOne("role", builder.Eq{"id": user.RoleID}, &role); f {
				perms = rbac.Split(role.Permissions)
			}
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("role_id", user.RoleID)
		c.Set("permissions", perms)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// AuthenticateToken 校验 JWT 并返回 claims（供 WS 等场景复用，token 来自 query 参数）
func AuthenticateToken(tokenStr string) (jwt.MapClaims, error) {
	if tokenStr == "" {
		return nil, fmt.Errorf("empty token")
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.GetString("jwt", "secret")), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// HasPermission 判断当前请求是否拥有指定权限
func HasPermission(c *gin.Context, perm string) bool {
	return rbac.Has(c.GetStringSlice("permissions"), perm)
}