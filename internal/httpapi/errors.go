package httpapi

import (
	"errors"
	"net/http"

	"designreview/internal/domain"
	"github.com/gin-gonic/gin"
)

type ErrorEnvelope struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	FieldErrors []domain.FieldError `json:"field_errors"`
	RequestID   string              `json:"request_id"`
}

func writeError(ctx *gin.Context, err error) {
	status, envelope := mapError(err, requestID(ctx))
	ctx.AbortWithStatusJSON(status, envelope)
}

func mapError(err error, requestID string) (int, ErrorEnvelope) {
	envelope := ErrorEnvelope{Code: "internal_error", Message: "request could not be completed", FieldErrors: []domain.FieldError{}, RequestID: requestID}
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		envelope.Code = "validation_failed"
		envelope.Message = validation.Message
		envelope.FieldErrors = append([]domain.FieldError(nil), validation.Fields...)
		return http.StatusUnprocessableEntity, envelope
	case errors.Is(err, domain.ErrNotFound):
		envelope.Code = "not_found"
		envelope.Message = "resource was not found"
		return http.StatusNotFound, envelope
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrAlreadyExists):
		envelope.Code = "conflict"
		envelope.Message = "resource changed or already exists"
		return http.StatusConflict, envelope
	case errors.Is(err, domain.ErrForbidden):
		envelope.Code = "forbidden"
		envelope.Message = "operation is not permitted"
		return http.StatusForbidden, envelope
	case errors.Is(err, domain.ErrInvalidState):
		envelope.Code = "invalid_state"
		envelope.Message = "resource state does not allow this operation"
		return http.StatusConflict, envelope
	case errors.Is(err, domain.ErrInvalidArgument):
		envelope.Code = "invalid_argument"
		envelope.Message = "request contains invalid values"
		return http.StatusBadRequest, envelope
	case errors.Is(err, domain.ErrCanceled):
		envelope.Code = "request_canceled"
		envelope.Message = "request was canceled"
		return http.StatusRequestTimeout, envelope
	default:
		return http.StatusInternalServerError, envelope
	}
}
