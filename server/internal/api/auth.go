package api

import (
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"gateway-manager/pkg/config"
	"gateway-manager/pkg/curd"
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

	user := config.GetString("auth", "admin_user")
	pass := config.GetString("auth", "admin_password")
	if subtle.ConstantTimeCompare([]byte(req.Username), []byte(user)) == 0 ||
		subtle.ConstantTimeCompare([]byte(req.Password), []byte(pass)) == 0 {
		curd.Error(c, 401, "用户名或密码错误")
		return
	}

	token, err := issueToken(user)
	if err != nil {
		curd.Error(c, 500, "生成令牌失败")
		return
	}
	curd.Success(c, gin.H{"token": token, "username": user})
}

// Me 当前登录用户
func Me(c *gin.Context) {
	curd.Success(c, gin.H{"username": c.GetString("username")})
}

func issueToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      username,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(config.GetString("jwt", "secret")))
}

// JWTAuth JWT 校验中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := AuthenticateToken(extractToken(c))
		if err != nil {
			curd.Unauthorized(c, "未登录或登录已过期")
			c.Abort()
			return
		}
		if username, ok := claims["username"].(string); ok {
			c.Set("username", username)
		}
		if sub, ok := claims["sub"].(string); ok {
			c.Set("user_id", sub)
		}
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