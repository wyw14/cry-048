package httpapi

import (
	"net/http"

	"designreview/internal/domain"
	"designreview/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	collaboration *service.CollaborationService
	reviews       *service.ReviewService
	publication   *service.PublicationService
	queries       *service.QueryService
	exports       *service.ExportService
}

func (handler *Handler) writeTransactionFailure(ctx *gin.Context, err error) {
	writeLegacyTransactionFailure(ctx, err)
}

func NewHandler(collaboration *service.CollaborationService, reviews *service.ReviewService, publication *service.PublicationService, queries *service.QueryService, exports *service.ExportService) *Handler {
	return &Handler{collaboration: collaboration, reviews: reviews, publication: publication, queries: queries, exports: exports}
}

func (handler *Handler) EditAnnotation(ctx *gin.Context) {
	var request editAnnotationRequest
	if err := bindJSON(ctx, &request); err != nil {
		writeError(ctx, err)
		return
	}
	value, err := handler.collaboration.EditAnnotation(ctx.Request.Context(), service.EditAnnotationCommand{AnnotationID: domain.ID(ctx.Param("annotationID")), Expected: domain.Revision(request.ExpectedVersion), Body: request.Body, Actor: request.Actor.domain()})
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, annotationDTO(value))
}

func (handler *Handler) TransitionAnnotation(ctx *gin.Context) {
	var request transitionAnnotationRequest
	if err := bindJSON(ctx, &request); err != nil {
		writeError(ctx, err)
		return
	}
	value, err := handler.collaboration.TransitionAnnotation(ctx.Request.Context(), service.TransitionAnnotationCommand{AnnotationID: domain.ID(ctx.Param("annotationID")), Expected: domain.Revision(request.ExpectedVersion), Target: domain.AnnotationState(request.State), Actor: request.Actor.domain()})
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, annotationDTO(value))
}

func (handler *Handler) CloseReview(ctx *gin.Context) {
	var request closeReviewRequest
	if err := bindJSON(ctx, &request); err != nil {
		writeError(ctx, err)
		return
	}
	recipients := make([]domain.ID, 0, len(request.Recipients))
	for _, recipient := range request.Recipients {
		recipients = append(recipients, domain.ID(recipient))
	}
	value, err := handler.reviews.Close(ctx.Request.Context(), service.CloseReviewCommand{RoundID: domain.ID(ctx.Param("reviewID")), Conclusion: request.Conclusion, Actor: request.Actor.domain(), Recipients: recipients})
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, reviewDTO(value))
}

func (handler *Handler) PublishVersion(ctx *gin.Context) {
	var request publishVersionRequest
	if err := bindJSON(ctx, &request); err != nil {
		writeError(ctx, err)
		return
	}
	value, err := handler.publication.Publish(ctx.Request.Context(), domain.ID(ctx.Param("versionID")), request.Actor.domain())
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"id": value.ID, "status": value.Status, "revision": value.Revision, "superseded_by": value.SupersededBy})
}

func (handler *Handler) ListAnnotations(ctx *gin.Context) {
	request, err := parseAnnotationQuery(ctx)
	if err != nil {
		writeError(ctx, err)
		return
	}
	page, err := handler.queries.ListAnnotations(ctx.Request.Context(), service.AnnotationQuery{VersionID: domain.ID(request.VersionID), State: domain.AnnotationState(request.State), SortBy: request.Sort, Descending: request.Descending, Page: request.Page, PageSize: request.PageSize})
	if err != nil {
		writeError(ctx, err)
		return
	}
	items := make([]annotationResponse, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, annotationDTO(value))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "page": page.Page, "page_size": page.PageSize, "total": page.Total, "request_id": requestID(ctx)})
}

func (handler *Handler) ExportAnnotations(ctx *gin.Context) {
	request, err := parseAnnotationQuery(ctx)
	if err != nil {
		writeError(ctx, err)
		return
	}
	key, err := handler.exports.ExportAnnotations(ctx.Request.Context(), service.AnnotationQuery{VersionID: domain.ID(request.VersionID), State: domain.AnnotationState(request.State), SortBy: request.Sort, Descending: request.Descending, PageSize: request.PageSize})
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, gin.H{"export_key": key, "request_id": requestID(ctx)})
}
