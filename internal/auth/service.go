package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/webdav-gateway/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// JWTClaims JWT令牌声明
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

// AuthService 认证服务
type AuthService struct {
	userRepo models.UserRepository
}

// NewService 创建认证服务
func NewService(userRepo models.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

// ValidateUser 验证用户凭据
func (s *AuthService) ValidateUser(username, password string) (string, error) {
	// 获取用户
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// 生成JWT令牌
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", err
	}

	return token, nil
}

// GenerateToken 生成JWT令牌
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// RevokeUserToken 撤销用户令牌
func (s *AuthService) RevokeUserToken(userID string) error {
	// 在实际实现中，这里可以将令牌加入黑名单
	// 可以存储在Redis中，并设置与令牌剩余有效期相同的TTL
	return nil
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}

// 错误定义
var (
	ErrInvalidCredentials = Error("invalid username or password")
	ErrUserNotFound       = Error("user not found")
	ErrTokenExpired       = Error("token has expired")
)

type Error string

func (e Error) Error() string {
	return string(e)
}