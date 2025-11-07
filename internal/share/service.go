package share

import (
	"time"

	"github.com/google/uuid"
	"github.com/webdav-gateway/internal/models"
)

// Service 共享服务
type Service struct {
	shareRepo models.ShareRepository
}

// NewService 创建共享服务
func NewService(shareRepo models.ShareRepository) *Service {
	return &Service{
		shareRepo: shareRepo,
	}
}

// CreateShareRequest 创建共享请求
type CreateShareRequest struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	Password     string     `json:"password,omitempty"`
	ExpiresAt    *int64     `json:"expires_at,omitempty"`
	MaxDownloads int        `json:"max_downloads,omitempty"`
	CreatedBy    string     `json:"created_by"`
}

// CreateShare 创建共享
func (s *Service) CreateShare(req CreateShareRequest) (*models.Share, error) {
	// 验证请求
	if req.Name == "" {
		return nil, ErrInvalidRequest
	}
	if req.Path == "" {
		return nil, ErrInvalidRequest
	}
	if req.CreatedBy == "" {
		return nil, ErrInvalidRequest
	}

	// 检查名称是否已存在
	if _, err := s.shareRepo.GetByName(req.Name); err == nil {
		return nil, ErrShareNameExists
	}

	// 创建共享
	share := &models.Share{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Path:         req.Path,
		Password:     req.Password,
		MaxDownloads: req.MaxDownloads,
		CreatedBy:    req.CreatedBy,
		CreatedAt:    time.Now(),
	}

	if req.ExpiresAt != nil {
		expiresAt := time.Unix(*req.ExpiresAt, 0)
		share.ExpiresAt = &expiresAt
	}

	if err := s.shareRepo.Create(share); err != nil {
		return nil, err
	}

	return share, nil
}

// GetUserShares 获取用户的共享列表
func (s *Service) GetUserShares(userID string, page, limit int) ([]models.Share, int, error) {
	shares, err := s.shareRepo.GetByUserID(userID)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.shareRepo.CountByUser(userID)
	if err != nil {
		return nil, 0, err
	}

	// 分页处理（简化实现）
	start := (page - 1) * limit
	end := start + limit
	if start >= len(shares) {
		return []models.Share{}, total, nil
	}
	if end > len(shares) {
		end = len(shares)
	}

	return shares[start:end], total, nil
}

// GetShare 获取共享
func (s *Service) GetShare(id string) (*models.Share, error) {
	share, err := s.shareRepo.GetByID(id)
	if err != nil {
		return nil, ErrShareNotFound
	}

	// 检查是否过期
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, ErrShareExpired
	}

	// 检查是否激活
	if !share.IsActive {
		return nil, ErrShareInactive
	}

	// 检查下载次数限制
	if share.MaxDownloads > 0 && share.DownloadCount >= share.MaxDownloads {
		return nil, ErrShareDownloadLimitExceeded
	}

	return share, nil
}

// UpdateShareRequest 更新共享请求
type UpdateShareRequest struct {
	Name         string `json:"name,omitempty"`
	Password     string `json:"password,omitempty"`
	ExpiresAt    *int64 `json:"expires_at,omitempty"`
	MaxDownloads int    `json:"max_downloads,omitempty"`
}

// UpdateShare 更新共享
func (s *Service) UpdateShare(id, userID string, req UpdateShareRequest) (*models.Share, error) {
	share, err := s.shareRepo.GetByID(id)
	if err != nil {
		return nil, ErrShareNotFound
	}

	// 检查权限
	if share.CreatedBy != userID {
		return nil, ErrUnauthorized
	}

	// 更新字段
	if req.Name != "" {
		share.Name = req.Name
	}
	if req.Password != "" {
		share.Password = req.Password
	}
	if req.ExpiresAt != nil {
		expiresAt := time.Unix(*req.ExpiresAt, 0)
		share.ExpiresAt = &expiresAt
	}
	if req.MaxDownloads >= 0 {
		share.MaxDownloads = req.MaxDownloads
	}

	if err := s.shareRepo.Update(share); err != nil {
		return nil, err
	}

	return share, nil
}

// DeleteShare 删除共享
func (s *Service) DeleteShare(id, userID string) error {
	share, err := s.shareRepo.GetByID(id)
	if err != nil {
		return ErrShareNotFound
	}

	// 检查权限
	if share.CreatedBy != userID {
		return ErrUnauthorized
	}

	return s.shareRepo.Delete(id)
}

// 错误定义
var (
	ErrInvalidRequest              = Error("invalid request")
	ErrShareNameExists            = Error("share name already exists")
	ErrShareNotFound              = Error("share not found")
	ErrShareExpired               = Error("share has expired")
	ErrShareInactive              = Error("share is inactive")
	ErrShareDownloadLimitExceeded = Error("share download limit exceeded")
	ErrUnauthorized               = Error("unauthorized")
)

type Error string

func (e Error) Error() string {
	return string(e)
}