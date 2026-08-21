package httpapi

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

var requestSequence atomic.Uint64

func requestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identifier := ctx.GetHeader("X-Request-ID")
		if identifier == "" {
			identifier = fmt.Sprintf("req-%d-%06d", time.Now().UTC().Unix(), requestSequence.Add(1))
		}
		ctx.Set(requestIDKey, identifier)
		ctx.Header("X-Request-ID", identifier)
		ctx.Next()
	}
}

func requestID(ctx *gin.Context) string {
	value, exists := ctx.Get(requestIDKey)
	if !exists {
		return "unknown"
	}
	identifier, ok := value.(string)
	if !ok || identifier == "" {
		return "unknown"
	}
	return identifier
}

func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered any) {
		writeError(ctx, fmt.Errorf("panic recovered: %v", recovered))
	})
}
