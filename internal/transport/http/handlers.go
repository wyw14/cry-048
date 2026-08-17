package http

import (
	"mime"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/cry048/design-review-platform/internal/application"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/middleware"
	"github.com/cry048/design-review-platform/internal/service"
)

// Handlers bundles all HTTP handlers and the validator.
type Handlers struct {
	Projects    *service.ProjectService
	Annotations *service.AnnotationService
	Reviews     *service.ReviewService
	Notifier    *service.NotifierService
	Audit       *service.AuditService
	Export      *service.ExportService
	Validator   *validator.Validate
}

// === healthz / readyz ===

func (h *Handlers) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handlers) Readyz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// === projects ===

type createProjectReq struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name" validate:"required,min=1,max=128"`
	Description string `json:"description,omitempty"`
}

func (h *Handlers) CreateProject(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	p, err := h.Projects.CreateProject(ctx, application.CreateProjectInput{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, p)
}

func (h *Handlers) GetProject(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	p, err := h.Projects.GetProject(ctx, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, p)
}

func (h *Handlers) ListProjects(c *gin.Context) {
	filter := application.ProjectFilter{
		Status: c.Query("status"),
		Name:   c.Query("q"),
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, total, err := h.Projects.ListProjects(ctx, filter)
	if err != nil {
		handleError(c, err)
		return
	}
	pg := parsePagination(c)
	writeList(c, items, total, pg.Page, pg.PageSize)
}

func (h *Handlers) ArchiveProject(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	if err := h.Projects.ArchiveProject(ctx, c.Param("id"), middleware.GetActorID(c)); err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"id": c.Param("id"), "status": "archived"})
}

type createBoardReq struct {
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"project_id" validate:"required"`
	Name      string `json:"name" validate:"required,min=1,max=128"`
	Width     int    `json:"width" validate:"required,min=1,max=100000"`
	Height    int    `json:"height" validate:"required,min=1,max=100000"`
}

func (h *Handlers) CreateBoard(c *gin.Context) {
	var req createBoardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	b, err := h.Projects.CreateBoard(ctx, application.CreateBoardInput{
		ID:        req.ID,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Width:     req.Width,
		Height:    req.Height,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, b)
}

func (h *Handlers) GetBoard(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	b, err := h.Projects.GetBoard(ctx, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, b)
}

func (h *Handlers) ListBoards(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Query("project_id")
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Projects.ListBoards(ctx, projectID)
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"items": items})
}

func (h *Handlers) CloseBoard(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	if err := h.Projects.CloseBoard(ctx, c.Param("id"), middleware.GetActorID(c)); err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"id": c.Param("id"), "status": "closed"})
}

type createVersionReq struct {
	ID         string `json:"id,omitempty"`
	BoardID    string `json:"board_id" validate:"required"`
	Number     int    `json:"number" validate:"required,min=1"`
	Label      string `json:"label" validate:"max=128"`
	PreviewKey string `json:"preview_key" validate:"required"`
	Notes      string `json:"notes,omitempty"`
	CreatedBy  string `json:"created_by,omitempty"`
}

func (h *Handlers) CreateVersion(c *gin.Context) {
	var req createVersionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	if req.CreatedBy == "" {
		req.CreatedBy = middleware.GetActorID(c)
	}
	v, err := h.Projects.CreateVersion(ctx, application.CreateVersionInput{
		ID:         req.ID,
		BoardID:    req.BoardID,
		Number:     req.Number,
		Label:      req.Label,
		PreviewKey: req.PreviewKey,
		Notes:      req.Notes,
		CreatedBy:  req.CreatedBy,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, v)
}

func (h *Handlers) GetVersion(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	v, err := h.Projects.GetVersion(ctx, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, v)
}

func (h *Handlers) ListVersions(c *gin.Context) {
	boardID := c.Param("id")
	if boardID == "" {
		boardID = c.Query("board_id")
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Projects.ListVersions(ctx, boardID)
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"items": items})
}

func (h *Handlers) PublishVersion(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	v, err := h.Projects.PublishVersion(ctx, c.Param("id"), middleware.GetActorID(c))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, v)
}

type addMemberReq struct {
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"project_id" validate:"required"`
	UserID    string `json:"user_id" validate:"required"`
	Role      string `json:"role" validate:"required,oneof=owner editor viewer"`
}

func (h *Handlers) AddMember(c *gin.Context) {
	var req addMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	m, err := h.Projects.AddMember(ctx, application.AddMemberInput{
		ID:        req.ID,
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		Role:      req.Role,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, m)
}

func (h *Handlers) ListMembers(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Query("project_id")
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Projects.ListMembers(ctx, projectID)
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"items": items})
}

func (h *Handlers) RemoveMember(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	if err := h.Projects.RemoveMember(ctx, c.Param("id"), middleware.GetActorID(c)); err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"id": c.Param("id"), "removed": true})
}

// === annotations ===

type createAnnotationReq struct {
	ID         string             `json:"id,omitempty"`
	ProjectID  string             `json:"project_id" validate:"required"`
	BoardID    string             `json:"board_id" validate:"required"`
	VersionID  string             `json:"version_id" validate:"required"`
	Title      string             `json:"title" validate:"required,min=1,max=200"`
	Body       string             `json:"body,omitempty"`
	ReporterID string             `json:"reporter_id,omitempty"`
	Point      *canvas.Coordinate `json:"point,omitempty"`
	Region     *canvas.Region     `json:"region,omitempty"`
	Priority   string             `json:"priority,omitempty" validate:"omitempty,oneof=low normal high critical"`
	AssigneeID string             `json:"assignee_id,omitempty"`
	DueAt      *time.Time         `json:"due_at,omitempty"`
}

func (h *Handlers) CreateAnnotation(c *gin.Context) {
	var req createAnnotationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	if req.ReporterID == "" {
		req.ReporterID = middleware.GetActorID(c)
	}
	if req.Point == nil && req.Region == nil {
		writeError(c, "VALIDATION", "批注必须包含点或区域", nil)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.Create(ctx, application.CreateAnnotationInput{
		ID:         req.ID,
		ProjectID:  req.ProjectID,
		BoardID:    req.BoardID,
		VersionID:  req.VersionID,
		Title:      req.Title,
		Body:       req.Body,
		ReporterID: req.ReporterID,
		Point:      req.Point,
		Region:     req.Region,
		Priority:   req.Priority,
		AssigneeID: req.AssigneeID,
		DueAt:      req.DueAt,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, a)
}

func (h *Handlers) GetAnnotation(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.Get(ctx, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

func (h *Handlers) ListAnnotations(c *gin.Context) {
	pg := parsePagination(c)
	filter := application.AnnotationFilter{
		ProjectID:  c.Query("project_id"),
		BoardID:    c.Query("board_id"),
		VersionID:  c.Query("version_id"),
		Status:     c.Query("status"),
		Priority:   c.Query("priority"),
		AssigneeID: c.Query("assignee_id"),
		ReporterID: c.Query("reporter_id"),
		Page:       pg.Page,
		PageSize:   pg.PageSize,
		SortBy:     pg.SortBy,
		SortDesc:   pg.SortDesc,
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, total, err := h.Annotations.List(ctx, filter)
	if err != nil {
		handleError(c, err)
		return
	}
	writeList(c, items, total, pg.Page, pg.PageSize)
}

func (h *Handlers) SearchAnnotations(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		writeError(c, "VALIDATION", "搜索关键词不能为空", nil)
		return
	}
	limit := atoiDefault(c.Query("limit"), 20)
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Annotations.Search(ctx, q, limit)
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"items": items, "count": len(items)})
}

func (h *Handlers) ListMyTodos(c *gin.Context) {
	userID := middleware.GetActorID(c)
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Annotations.ListForUser(ctx, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"items": items, "count": len(items)})
}

type addReplyReq struct {
	Body string `json:"body" validate:"required,min=1"`
}

func (h *Handlers) AddReply(c *gin.Context) {
	var req addReplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.AddReply(ctx, application.AddReplyInput{
		AnnotationID: c.Param("id"),
		AuthorID:     middleware.GetActorID(c),
		Body:         req.Body,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, a)
}

type setAssigneeReq struct {
	AssigneeID string `json:"assignee_id" validate:"required"`
}

func (h *Handlers) SetAssignee(c *gin.Context) {
	var req setAssigneeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.SetAssignee(ctx, c.Param("id"), req.AssigneeID, middleware.GetActorID(c))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

type setDueReq struct {
	DueAt *time.Time `json:"due_at"`
}

func (h *Handlers) SetDue(c *gin.Context) {
	var req setDueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.SetDueAt(ctx, c.Param("id"), req.DueAt, middleware.GetActorID(c))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

type setPriorityReq struct {
	Priority string `json:"priority" validate:"required,oneof=low normal high critical"`
}

func (h *Handlers) SetPriority(c *gin.Context) {
	var req setPriorityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.SetPriority(ctx, c.Param("id"), req.Priority, middleware.GetActorID(c))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

func (h *Handlers) RequestReview(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.RequestReview(ctx, application.RequestReviewInput{
		AnnotationID: c.Param("id"),
		ActorID:      middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

func (h *Handlers) ResolveAnnotation(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.Resolve(ctx, application.ResolveAnnotationInput{
		AnnotationID: c.Param("id"),
		ActorID:      middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

type reopenReq struct {
	Reason string `json:"reason,omitempty"`
}

func (h *Handlers) ReopenAnnotation(c *gin.Context) {
	var req reopenReq
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		// tolerate empty body
		_ = err
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.Reopen(ctx, application.ReopenAnnotationInput{
		AnnotationID: c.Param("id"),
		ActorID:      middleware.GetActorID(c),
		Reason:       req.Reason,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

func (h *Handlers) CloseAnnotation(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.Close(ctx, application.CloseAnnotationInput{
		AnnotationID: c.Param("id"),
		ActorID:      middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, a)
}

type addAttachmentReq struct {
	Filename   string `json:"filename" validate:"required"`
	MediaType  string `json:"media_type" validate:"required"`
	Size       int64  `json:"size" validate:"required,min=1"`
	StorageKey string `json:"storage_key" validate:"required"`
}

func (h *Handlers) AddAttachment(c *gin.Context) {
	var req addAttachmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	a, err := h.Annotations.AddAttachment(ctx, application.AddAttachmentInput{
		AnnotationID: c.Param("id"),
		Filename:     req.Filename,
		MediaType:    req.MediaType,
		Size:         req.Size,
		StorageKey:   req.StorageKey,
		UploadedBy:   middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, a)
}

type migrateAnchorsReq struct {
	FromVersionID string `json:"from_version_id" validate:"required"`
	ToVersionID   string `json:"to_version_id" validate:"required"`
	Reason        string `json:"reason,omitempty"`
}

func (h *Handlers) MigrateAnchors(c *gin.Context) {
	var req migrateAnchorsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	n, err := h.Annotations.MigrateAnchors(ctx, application.MigrateAnchorsInput{
		FromVersionID: req.FromVersionID,
		ToVersionID:   req.ToVersionID,
		By:            middleware.GetActorID(c),
		Reason:        req.Reason,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"migrated_count": n})
}

// === reviews ===

type createRoundReq struct {
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"project_id" validate:"required"`
	BoardID   string `json:"board_id" validate:"required"`
	Title     string `json:"title" validate:"required,min=1,max=200"`
}

func (h *Handlers) CreateRound(c *gin.Context) {
	var req createRoundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	rd, err := h.Reviews.CreateRound(ctx, application.CreateRoundInput{
		ID:        req.ID,
		ProjectID: req.ProjectID,
		BoardID:   req.BoardID,
		Title:     req.Title,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, rd)
}

func (h *Handlers) GetRound(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	rd, err := h.Reviews.GetRound(ctx, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, rd)
}

func (h *Handlers) ListRounds(c *gin.Context) {
	boardID := c.Param("boardID")
	if boardID == "" {
		boardID = c.Query("board_id")
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Reviews.ListRounds(ctx, boardID)
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, gin.H{"items": items})
}

type snapshotReq struct {
	ProjectID string `json:"project_id" validate:"required"`
	VersionID string `json:"version_id" validate:"required"`
	BoardID   string `json:"board_id,omitempty"`
}

func (h *Handlers) CreateSnapshot(c *gin.Context) {
	var req snapshotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	s, err := h.Reviews.CreateSnapshot(ctx, application.SnapshotInput{
		RoundID:   c.Param("id"),
		BoardID:   req.BoardID,
		VersionID: req.VersionID,
		ProjectID: req.ProjectID,
		By:        middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeCreated(c, s)
}

type conclusionReq struct {
	Conclusion     string `json:"conclusion" validate:"required,min=1"`
	Recommendation string `json:"recommendation" validate:"required,oneof=approve approve_with_conditions reject defer"`
}

func (h *Handlers) SetConclusion(c *gin.Context) {
	var req conclusionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, err)
		return
	}
	if err := h.Validator.Struct(req); err != nil {
		handleError(c, err)
		return
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	rd, err := h.Reviews.SetConclusion(ctx, application.SetConclusionInput{
		RoundID:        c.Param("id"),
		Conclusion:     req.Conclusion,
		Recommendation: req.Recommendation,
		ActorID:        middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, rd)
}

func (h *Handlers) CloseRound(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	rd, err := h.Reviews.CloseRound(ctx, application.CloseRoundInput{
		RoundID: c.Param("id"),
		ActorID: middleware.GetActorID(c),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	writeOK(c, rd)
}

// === notifications ===

func (h *Handlers) ListNotifications(c *gin.Context) {
	userID := middleware.GetActorID(c)
	unreadOnly := c.Query("unread") == "true"
	items := h.Notifier.List(userID, unreadOnly)
	writeList(c, items, len(items), 1, len(items))
}

func (h *Handlers) MarkNotificationRead(c *gin.Context) {
	userID := middleware.GetActorID(c)
	ok := h.Notifier.MarkRead(userID, c.Param("id"))
	if !ok {
		writeError(c, "NOT_FOUND", "通知不存在", nil)
		return
	}
	writeOK(c, gin.H{"id": c.Param("id"), "read": true})
}

func (h *Handlers) UnreadCount(c *gin.Context) {
	userID := middleware.GetActorID(c)
	writeOK(c, gin.H{"unread": h.Notifier.UnreadCount(userID)})
}

// === audit ===

func (h *Handlers) ListAudit(c *gin.Context) {
	from := time.Time{}
	to := time.Time{}
	if s := c.Query("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = t
		}
	}
	if s := c.Query("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = t
		}
	}
	filter := application.AuditFilter{
		ActorID:    c.Query("actor_id"),
		EntityType: c.Query("entity_type"),
		EntityID:   c.Query("entity_id"),
		From:       from,
		To:         to,
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	items, err := h.Audit.Query(ctx, filter)
	if err != nil {
		handleError(c, err)
		return
	}
	writeList(c, items, len(items), 1, len(items))
}

// === export ===

func (h *Handlers) ExportAnnotations(c *gin.Context) {
	pg := parsePagination(c)
	filter := application.AnnotationFilter{
		ProjectID: c.Query("project_id"),
		BoardID:   c.Query("board_id"),
		VersionID: c.Query("version_id"),
		Status:    c.Query("status"),
		Page:      pg.Page,
		PageSize:  pg.PageSize,
	}
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	csv, err := h.Export.ExportAnnotationsCSV(ctx, filter)
	if err != nil {
		handleError(c, err)
		return
	}
	writeCSVDownload(c, "annotations.csv", csv)
}

func (h *Handlers) ExportReviewMinutes(c *gin.Context) {
	ctx, cancel := withTimeout(c.Request.Context())
	defer cancel()
	csv, err := h.Export.ExportReviewMinutesCSV(ctx, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	writeCSVDownload(c, "review_minutes.csv", csv)
}

func writeCSVDownload(c *gin.Context, filename, payload string) {
	disposition := mime.FormatMediaType("attachment", map[string]string{
		"filename": filename,
	})
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(payload))
}
