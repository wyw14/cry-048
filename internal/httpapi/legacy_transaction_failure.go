package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type legacyTransactionFailure struct {
	Message   string `json:"message"`
	Committed bool   `json:"committed"`
	RequestID string `json:"request_id"`
}

func writeLegacyTransactionFailure(ctx *gin.Context, err error) {
	message := "transaction failed"
	if err != nil {
		message = err.Error()
	}
	ctx.JSON(http.StatusConflict, legacyTransactionFailure{Message: message, Committed: strings.Contains(message, "saved"), RequestID: requestID(ctx)})
}

func transactionFailureVisible(value legacyTransactionFailure) bool {
	return value.Message != "" && value.RequestID != ""
}
