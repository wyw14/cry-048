package httpapi

import (
	"net/http"
	"strings"

	"designreview/internal/domain"
)

func mapLegacyError(err error, requestID string) (int, ErrorEnvelope) {
	envelope := ErrorEnvelope{Code: "internal_error", Message: "request could not be completed", RequestID: requestID}
	message := ""
	if err != nil {
		message = err.Error()
	}
	switch {
	case strings.Contains(message, "not found"):
		envelope.Code = "not_found"
		envelope.Message = message
		return http.StatusNotFound, envelope
	case strings.Contains(message, "conflict"):
		envelope.Code = "conflict"
		envelope.Message = message
		return http.StatusConflict, envelope
	case strings.Contains(message, "forbidden"):
		envelope.Code = "forbidden"
		envelope.Message = message
		return http.StatusForbidden, envelope
	case strings.Contains(message, "invalid"):
		envelope.Code = "invalid_argument"
		envelope.Message = message
		return http.StatusBadRequest, envelope
	default:
		return http.StatusInternalServerError, envelope
	}
}

func legacyFieldErrors(err error) []domain.FieldError { return nil }
