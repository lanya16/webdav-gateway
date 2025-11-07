package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/webdav-gateway/internal/auth"
	"github.com/webdav-gateway/internal/models"
	"github.com/webdav-gateway/internal/storage"
)

func handleRegister(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := authService.Register(c.Request.Context(), &req)
		if err != nil {
			if err == auth.ErrUserExists {
				c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
			return
		}

		c.JSON(http.StatusCreated, user)
	}
}

func handleLogin(authService *auth.Service, storageService *storage.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := authService.Login(c.Request.Context(), &req)
		if err != nil {
			if err == auth.ErrInvalidCredentials {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
			return
		}

		// Ensure user bucket exists
		if err := storageService.EnsureBucket(c.Request.Context(), resp.User.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to setup storage"})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func handleGetMe(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("userID")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		user, err := authService.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}