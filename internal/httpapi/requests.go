package httpapi

import (
	"strconv"
	"strings"
	"time"

	"designreview/internal/domain"
	"github.com/gin-gonic/gin"
)

type editAnnotationRequest struct {
	ExpectedVersion int64        `json:"expected_version"`
	Body            string       `json:"body"`
	Actor           actorRequest `json:"actor"`
}
type actorRequest struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
type transitionAnnotationRequest struct {
	ExpectedVersion int64        `json:"expected_version"`
	State           string       `json:"state"`
	Actor           actorRequest `json:"actor"`
}
type closeReviewRequest struct {
	Conclusion string       `json:"conclusion"`
	Actor      actorRequest `json:"actor"`
	Recipients []string     `json:"recipients"`
}
type publishVersionRequest struct {
	Actor actorRequest `json:"actor"`
}

func (request actorRequest) domain() domain.Actor {
	return domain.Actor{ID: domain.ID(strings.TrimSpace(request.ID)), Email: strings.TrimSpace(request.Email), Name: strings.TrimSpace(request.Name)}
}

func bindJSON(ctx *gin.Context, destination any) error {
	if err := ctx.ShouldBindJSON(destination); err != nil {
		return domain.NewValidationError("invalid JSON body", domain.FieldError{Field: "body", Message: err.Error()})
	}
	return nil
}

func parseAnnotationQuery(ctx *gin.Context) (query AnnotationQueryDTO, err error) {
	query.VersionID = strings.TrimSpace(ctx.Query("version_id"))
	query.State = strings.TrimSpace(ctx.Query("state"))
	query.Sort = strings.TrimSpace(ctx.DefaultQuery("sort", "id"))
	query.Descending = ctx.Query("direction") == "desc"
	query.Page, err = parsePositive(ctx.DefaultQuery("page", "1"), "page")
	if err != nil {
		return AnnotationQueryDTO{}, err
	}
	query.PageSize, err = parsePositive(ctx.DefaultQuery("page_size", "25"), "page_size")
	if err != nil {
		return AnnotationQueryDTO{}, err
	}
	return query, nil
}

type AnnotationQueryDTO struct {
	VersionID  string
	State      string
	Sort       string
	Descending bool
	Page       int
	PageSize   int
}

func parsePositive(value, field string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 {
		return 0, domain.NewValidationError("invalid query", domain.FieldError{Field: field, Message: "must be a positive integer"})
	}
	return number, nil
}

type annotationResponse struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	BoardID    string `json:"board_id"`
	VersionID  string `json:"version_id"`
	NodeKey    string `json:"node_key"`
	Body       string `json:"body"`
	State      string `json:"state"`
	Revision   int64  `json:"revision"`
	ReplyCount int    `json:"reply_count"`
}

func annotationDTO(value domain.Annotation) annotationResponse {
	return annotationResponse{ID: value.ID.String(), ProjectID: value.ProjectID.String(), BoardID: value.BoardID.String(), VersionID: value.Anchor.VersionID.String(), NodeKey: value.Anchor.NodeKey, Body: value.Body, State: string(value.State), Revision: int64(value.Revision), ReplyCount: len(value.Replies)}
}

type reviewResponse struct {
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	Conclusion      string     `json:"conclusion"`
	Revision        int64      `json:"revision"`
	ClosedAt        *time.Time `json:"closed_at"`
	AnnotationCount int        `json:"annotation_count"`
}

func reviewDTO(value domain.ReviewRound) reviewResponse {
	count := 0
	if value.Snapshot != nil {
		count = len(value.Snapshot.AnnotationIDs)
	}
	return reviewResponse{ID: value.ID.String(), Status: string(value.Status), Conclusion: value.Conclusion, Revision: int64(value.Revision), ClosedAt: value.ClosedAt, AnnotationCount: count}
}
