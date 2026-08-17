// Package http provides HTTP transport for the design-review platform.
package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/domain/project"
	"github.com/cry048/design-review-platform/internal/domain/review"
	"github.com/cry048/design-review-platform/internal/middleware"
)

// APIError is a stable error envelope.
type APIError struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	FieldErrors []FieldError `json:"field_errors,omitempty"`
	RequestID   string       `json:"request_id"`
}

type FieldError struct {
	Field  string `json:"field"`
	Tag    string `json:"tag"`
	Value  string `json:"value,omitempty"`
	Reason string `json:"reason"`
}

func (e APIError) Status() int {
	switch e.Code {
	case "NOT_FOUND":
		return http.StatusNotFound
	case "VALIDATION", "INVALID_PRIORITY", "INVALID_STATUS", "INVALID_TRANSITION", "INVALID_BOARD_SIZE", "INVALID_ROLE", "INVALID_RECOMMENDATION":
		return http.StatusBadRequest
	case "CONFLICT", "STALE_VERSION", "ALREADY_EXISTS":
		return http.StatusConflict
	case "FORBIDDEN":
		return http.StatusForbidden
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	}
	return http.StatusBadRequest
}

func writeError(c *gin.Context, code, msg string, fields []FieldError) {
	e := APIError{
		Code:        code,
		Message:     msg,
		FieldErrors: fields,
		RequestID:   middleware.GetRequestID(c),
	}
	c.AbortWithStatusJSON(e.Status(), gin.H{"error": e})
}

// errToAPIError maps domain errors to API errors.
func errToAPIError(err error) (string, string) {
	switch {
	case errors.Is(err, project.ErrProjectNotFound):
		return "NOT_FOUND", "项目不存在"
	case errors.Is(err, project.ErrBoardNotFound):
		return "NOT_FOUND", "画板不存在"
	case errors.Is(err, project.ErrVersionNotFound):
		return "NOT_FOUND", "版本不存在"
	case errors.Is(err, project.ErrMemberNotFound):
		return "NOT_FOUND", "成员不存在"
	case errors.Is(err, project.ErrMemberAlreadyExists):
		return "ALREADY_EXISTS", "成员已存在"
	case errors.Is(err, project.ErrInvalidRole):
		return "INVALID_ROLE", "无效的成员角色"
	case errors.Is(err, project.ErrInvalidBoardSize):
		return "INVALID_BOARD_SIZE", "无效的画布尺寸"
	case errors.Is(err, project.ErrVersionAlreadyExists):
		return "ALREADY_EXISTS", "版本号已存在"
	case errors.Is(err, project.ErrBoardClosed):
		return "CONFLICT", "画板已关闭"
	case errors.Is(err, project.ErrProjectArchived):
		return "CONFLICT", "项目已归档"
	case errors.Is(err, project.ErrVersionNumberInvalid):
		return "VALIDATION", "版本号必须为正整数"
	case errors.Is(err, annotation.ErrAnnotationNotFound):
		return "NOT_FOUND", "批注不存在"
	case errors.Is(err, annotation.ErrReplyNotFound):
		return "NOT_FOUND", "回复不存在"
	case errors.Is(err, annotation.ErrAttachmentNotFound):
		return "NOT_FOUND", "附件不存在"
	case errors.Is(err, annotation.ErrInvalidPriority):
		return "INVALID_PRIORITY", "无效的优先级"
	case errors.Is(err, annotation.ErrInvalidStatus):
		return "INVALID_STATUS", "无效的状态"
	case errors.Is(err, annotation.ErrInvalidTransition):
		return "INVALID_TRANSITION", "无效的状态流转"
	case errors.Is(err, annotation.ErrMustNotBeEmpty):
		return "VALIDATION", "字段不能为空"
	case errors.Is(err, annotation.ErrDueDateInPast):
		return "VALIDATION", "截止时间不能早于当前时间"
	case errors.Is(err, annotation.ErrAssigneeRequired):
		return "VALIDATION", "当前状态变更需要负责人"
	case errors.Is(err, annotation.ErrCannotResolveWithoutReply):
		return "INVALID_TRANSITION", "解决批注前必须至少有一条回复"
	case errors.Is(err, annotation.ErrCannotReopenUnresolved):
		return "INVALID_TRANSITION", "仅已解决的批注可重新打开"
	case errors.Is(err, annotation.ErrAttachmentTooLarge):
		return "VALIDATION", "附件过大"
	case errors.Is(err, annotation.ErrAttachmentTypeForbidden):
		return "VALIDATION", "附件类型不允许"
	case errors.Is(err, annotation.ErrStaleVersion):
		return "STALE_VERSION", "数据已被他人修改，请刷新后重试"
	case errors.Is(err, review.ErrRoundNotFound):
		return "NOT_FOUND", "评审轮次不存在"
	case errors.Is(err, review.ErrRoundClosed):
		return "CONFLICT", "评审轮次已关闭"
	case errors.Is(err, review.ErrRoundAlreadyClosed):
		return "CONFLICT", "评审轮次已经关闭"
	case errors.Is(err, review.ErrSnapshotNotFound):
		return "NOT_FOUND", "快照不存在"
	case errors.Is(err, review.ErrSummaryEmpty):
		return "VALIDATION", "汇总结论不能为空"
	case errors.Is(err, review.ErrInvalidRecommendation):
		return "INVALID_RECOMMENDATION", "无效的发布建议"
	case errors.Is(err, canvas.ErrInvalidCoordinate):
		return "VALIDATION", "坐标超出画布范围"
	case errors.Is(err, canvas.ErrInvalidRegion):
		return "VALIDATION", "区域超出画布范围或尺寸非正"
	}
	return "INTERNAL", err.Error()
}

func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make([]FieldError, 0, len(verrs))
		for _, fe := range verrs {
			fields = append(fields, FieldError{
				Field:  fe.Field(),
				Tag:    fe.Tag(),
				Value:  fmtVal(fe.Value()),
				Reason: reasonFor(fe),
			})
		}
		writeError(c, "VALIDATION", "请求参数校验失败", fields)
		return
	}
	code, msg := errToAPIError(err)
	writeError(c, code, msg, nil)
}

func fmtVal(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(toString(v))
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func reasonFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "该字段必填"
	case "min":
		return "小于最小值"
	case "max":
		return "大于最大值"
	case "oneof":
		return "取值不在允许范围"
	case "len":
		return "长度不符"
	}
	return fe.Tag()
}

// pagination params shared across list endpoints.
type paginationParams struct {
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}

func parsePagination(c *gin.Context) paginationParams {
	p := paginationParams{
		Page:     atoiDefault(c.Query("page"), 1),
		PageSize: atoiDefault(c.Query("page_size"), 20),
		SortBy:   c.Query("sort_by"),
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	p.SortDesc = c.Query("order") == "desc"
	return p
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// response wrappers
func writeCreated(c *gin.Context, body interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": body})
}

func writeOK(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": body})
}

func writeList(c *gin.Context, items interface{}, total int, page, pageSize int) {
	c.JSON(http.StatusOK, gin.H{
		"data":       items,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"request_id": middleware.GetRequestID(c),
	})
}

// for tests that need an empty context with timeout
var contextTimeout = 5 * time.Second

func withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, contextTimeout)
}
