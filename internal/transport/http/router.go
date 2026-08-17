package http

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/cry048/design-review-platform/internal/config"
	domainaudit "github.com/cry048/design-review-platform/internal/domain/audit"
	"github.com/cry048/design-review-platform/internal/middleware"
)

// NewRouter builds the gin engine with all routes registered.
func NewRouter(cfg *config.Config, logger *zap.Logger, h *Handlers, auditLogger domainaudit.Logger) *gin.Engine {
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.ActorID(cfg.Runtime.DefaultUserID))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.CORS.AllowOrigins, cfg.CORS.AllowHeaders, cfg.CORS.AllowMethods))
	r.Use(middleware.AuditLog(logger, auditLogger))

	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)

	api := r.Group("/api/v1")
	{
		// projects
		api.POST("/projects", h.CreateProject)
		api.GET("/projects", h.ListProjects)
		api.GET("/projects/:id", h.GetProject)
		api.POST("/projects/:id/archive", h.ArchiveProject)
		api.GET("/projects/:id/boards", h.ListBoards)

		// boards
		api.POST("/boards", h.CreateBoard)
		api.GET("/boards/:id", h.GetBoard)
		api.POST("/boards/:id/close", h.CloseBoard)
		api.GET("/boards/:id/versions", h.ListVersions)

		// versions
		api.POST("/versions", h.CreateVersion)
		api.GET("/versions/:id", h.GetVersion)
		api.POST("/versions/:id/publish", h.PublishVersion)

		// members
		api.POST("/members", h.AddMember)
		api.GET("/members", h.ListMembers)
		api.DELETE("/members/:id", h.RemoveMember)

		// annotations
		api.POST("/annotations", h.CreateAnnotation)
		api.GET("/annotations", h.ListAnnotations)
		api.GET("/annotations/search", h.SearchAnnotations)
		api.GET("/annotations/my-todos", h.ListMyTodos)
		api.GET("/annotations/:id", h.GetAnnotation)
		api.POST("/annotations/:id/replies", h.AddReply)
		api.PATCH("/annotations/:id/assignee", h.SetAssignee)
		api.PATCH("/annotations/:id/due", h.SetDue)
		api.PATCH("/annotations/:id/priority", h.SetPriority)
		api.POST("/annotations/:id/request-review", h.RequestReview)
		api.POST("/annotations/:id/resolve", h.ResolveAnnotation)
		api.POST("/annotations/:id/reopen", h.ReopenAnnotation)
		api.POST("/annotations/:id/close", h.CloseAnnotation)
		api.POST("/annotations/:id/attachments", h.AddAttachment)
		api.POST("/annotations/migrate-anchors", h.MigrateAnchors)

		// reviews
		api.POST("/reviews/rounds", h.CreateRound)
		api.GET("/reviews/rounds/:id", h.GetRound)
		api.GET("/reviews/boards/:boardID/rounds", h.ListRounds)
		api.POST("/reviews/rounds/:id/snapshots", h.CreateSnapshot)
		api.POST("/reviews/rounds/:id/conclusion", h.SetConclusion)
		api.POST("/reviews/rounds/:id/close", h.CloseRound)

		// notifications
		api.GET("/notifications", h.ListNotifications)
		api.POST("/notifications/:id/read", h.MarkNotificationRead)
		api.GET("/notifications/unread-count", h.UnreadCount)

		// audit
		api.GET("/audit", h.ListAudit)

		// export
		api.GET("/export/annotations.csv", h.ExportAnnotations)
		api.GET("/export/reviews/:id/minutes.csv", h.ExportReviewMinutes)
	}
	return r
}
