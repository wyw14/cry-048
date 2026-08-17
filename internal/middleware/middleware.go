// Package middleware provides HTTP middlewares: request id, panic recovery, CORS, security headers, audit logging.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	domainaudit "github.com/cry048/design-review-platform/internal/domain/audit"
)

// ContextKey is a typed key for context values.
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	ActorIDKey   ContextKey = "actor_id"
)

// RequestID middleware adds an X-Request-ID header and stashes it in context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(string(RequestIDKey), id)
		c.Header("X-Request-ID", id)
		ctx := c.Request.Context()
		c.Request = c.Request.WithContext(context.WithValue(ctx, RequestIDKey, id))
		c.Next()
	}
}

// GetRequestID returns the request id from context.
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(string(RequestIDKey)); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ActorID injects a default user id into context for offline use.
func ActorID(defaultUserID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := c.GetHeader("X-User-ID")
		if actor == "" {
			actor = defaultUserID
		}
		c.Set(string(ActorIDKey), actor)
		ctx := c.Request.Context()
		c.Request = c.Request.WithContext(context.WithValue(ctx, ActorIDKey, actor))
		c.Next()
	}
}

// GetActorID returns the actor id from context.
func GetActorID(c *gin.Context) string {
	if v, ok := c.Get(string(ActorIDKey)); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "system"
}

// Recovery recovers from panics and returns a structured 500 error.
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					zap.Any("recover", rec),
					zap.String("request_id", GetRequestID(c)),
					zap.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":       "INTERNAL",
						"message":    "internal server error",
						"request_id": GetRequestID(c),
					},
				})
			}
		}()
		c.Next()
	}
}

// SecurityHeaders sets common security headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'")
		c.Next()
	}
}

// CORS returns a CORS middleware configured with the given origins/headers/methods.
func CORS(allowOrigins, allowHeaders, allowMethods []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowHeaders:     allowHeaders,
		AllowMethods:     allowMethods,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

// AuditLog logs each request to the audit logger (offline).
func AuditLog(logger *zap.Logger, auditLogger domainaudit.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		_ = auditLogger.Log(c.Request.Context(), domainaudit.Entry{
			ActorID:    GetActorID(c),
			Action:     fmt.Sprintf("http.%s.%s", c.Request.Method, strings.ToLower(c.Request.URL.Path)),
			EntityType: "http",
			EntityID:   GetRequestID(c),
			Before:     "",
			After:      fmt.Sprintf("status=%d dur=%s", c.Writer.Status(), duration),
			At:         start,
			RequestID:  GetRequestID(c),
		})
		logger.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("request_id", GetRequestID(c)),
			zap.String("actor_id", GetActorID(c)),
		)
	}
}
