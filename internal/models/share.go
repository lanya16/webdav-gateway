package models

import (
	"time"

	"github.com/google/uuid"
)

type FileShare struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	FilePath      string     `json:"file_path"`
	ShareToken    string     `json:"share_token"`
	ShareName     string     `json:"share_name"`
	PasswordHash  string     `json:"-"`
	ExpiresAt     *time.Time `json:"expires_at"`
	MaxDownloads  *int       `json:"max_downloads"`
	DownloadCount int        `json:"download_count"`
	Permissions   string     `json:"permissions"`
	CreatedAt     time.Time  `json:"created_at"`
}

type CreateShareRequest struct {
	FilePath     string `json:"file_path" binding:"required"`
	ShareName    string `json:"share_name"`
	Password     string `json:"password"`
	ExpiresIn    int    `json:"expires_in"` // hours
	MaxDownloads *int   `json:"max_downloads"`
	Permissions  string `json:"permissions"`
}

type CreateShareResponse struct {
	ShareURL   string     `json:"share_url"`
	ShareToken string     `json:"share_token"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type AccessShareRequest struct {
	Password string `json:"password"`
}
