package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/webdav-gateway/internal/auth"
	"github.com/webdav-gateway/internal/models"
	"github.com/webdav-gateway/internal/share"
	"github.com/webdav-gateway/internal/storage"
)

func handleCreateShare(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("userID")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		var req models.CreateShareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := shareService.CreateShare(c.Request.Context(), userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create share"})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

func handleListShares(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("userID")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		shares, err := shareService.ListUserShares(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shares"})
			return
		}

		c.JSON(http.StatusOK, shares)
	}
}

func handleDeleteShare(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("userID")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		shareID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid share id"})
			return
		}

		if err := shareService.DeleteShare(c.Request.Context(), shareID, userID); err != nil {
			if err == share.ErrShareNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete share"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func handleGetShare(shareService *share.Service, storageService *storage.Service, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")

		fileShare, err := shareService.GetShare(c.Request.Context(), token)
		if err != nil {
			if err == share.ErrShareNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get share"})
			return
		}

		// Return share info (without downloading the file)
		c.JSON(http.StatusOK, gin.H{
			"share_name":     fileShare.ShareName,
			"file_path":      fileShare.FilePath,
			"expires_at":     fileShare.ExpiresAt,
			"download_count": fileShare.DownloadCount,
			"max_downloads":  fileShare.MaxDownloads,
			"has_password":   fileShare.PasswordHash != "",
		})
	}
}

func handleAccessShare(shareService *share.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")

		var req models.AccessShareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fileShare, err := shareService.ValidateShareAccess(c.Request.Context(), token, req.Password)
		if err != nil {
			if err == share.ErrShareNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
				return
			}
			if err == share.ErrShareExpired {
				c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
				return
			}
			if err == share.ErrMaxDownloads {
				c.JSON(http.StatusForbidden, gin.H{"error": "maximum downloads reached"})
				return
			}
			if err == share.ErrInvalidPassword {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access share"})
			return
		}

		// Increment download count
		if err := shareService.IncrementDownloadCount(c.Request.Context(), fileShare.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update download count"})
			return
		}

		// Return download URL or file info
		c.JSON(http.StatusOK, gin.H{
			"message":    "access granted",
			"file_path":  fileShare.FilePath,
			"share_name": fileShare.ShareName,
		})
	}
}