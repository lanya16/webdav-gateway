package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		endTime := time.Now()
		latency := endTime.Sub(startTime)

		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		userID := c.GetString("userID")

		logger.WithFields(logrus.Fields{
			"status":   statusCode,
			"method":   method,
			"path":     path,
			"latency":  latency,
			"ip":       clientIP,
			"user_id":  userID,
		}).Info("request processed")
	}
}

func RecoveryMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithFields(logrus.Fields{
					"error": err,
					"path":  c.Request.URL.Path,
				}).Error("panic recovered")
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
