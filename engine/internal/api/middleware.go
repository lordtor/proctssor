package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a gin middleware for logging
func Logger(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Infow("request",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)
	}
}

// Recovery returns a gin middleware for panic recovery
func Recovery(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorw("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
				)
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}

// CORS returns a gin middleware for CORS
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestID returns a gin middleware that adds a request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}

// Timeout returns a gin middleware that adds a timeout to requests
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Note: For proper implementation, use http.TimeoutHandler
		// This is a simplified version
		c.Next()
	}
}

// RateLimiter returns a gin middleware for rate limiting
func RateLimiter(requestsPerSecond int) gin.HandlerFunc {
	// Simplified rate limiter
	// In production, use a proper rate limiting library
	return func(c *gin.Context) {
		// TODO: Implement proper rate limiting
		c.Next()
	}
}

// Auth returns a gin middleware for authentication
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Allow unauthenticated requests for now
			// In production, implement proper authentication
		}
		c.Next()
	}
}

// ValidateContentType returns a gin middleware that validates content type
func ValidateContentType(allowedTypes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			valid := false
			for _, t := range allowedTypes {
				if contentType == t || contentType == t+"; charset=utf-8" {
					valid = true
					break
				}
			}
			if !valid && contentType != "" {
				c.JSON(415, gin.H{"error": "Unsupported Content-Type"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// Metrics returns a gin middleware for collecting metrics
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		// TODO: Send metrics to Prometheus or similar
		_ = duration
	}
}
