package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/webdav-gateway/internal/auth"
)

func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check Authorization header
		authHeader := c.GetHeader("Authorization")
		
		if authHeader == "" {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Support both Bearer token and Basic auth
		var token string
		
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// For WebDAV clients, we'll use the username as token
			// This is a simplified approach - in production you might want to handle Basic auth properly
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		} else {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Set user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Depth, Destination, Overwrite")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Last-Modified, ETag")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func StorageQuotaMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check for PUT requests
		if c.Request.Method != "PUT" {
			c.Next()
			return
		}

		userIDStr := c.GetString("userID")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		user, err := authService.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Check if upload would exceed quota
		contentLength := c.Request.ContentLength
		if user.StorageUsed+contentLength > user.StorageQuota {
			c.JSON(http.StatusInsufficientStorage, gin.H{
				"error": "storage quota exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
