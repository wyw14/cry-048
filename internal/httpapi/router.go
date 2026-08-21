package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(requestIDMiddleware(), recoveryMiddleware())
	router.GET("/healthz", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	api := router.Group("/api/v1")
	api.PATCH("/annotations/:annotationID", handler.EditAnnotation)
	api.POST("/annotations/:annotationID/transitions", handler.TransitionAnnotation)
	api.POST("/reviews/:reviewID/close", handler.CloseReview)
	api.POST("/versions/:versionID/publish", handler.PublishVersion)
	api.GET("/annotations", handler.ListAnnotations)
	api.POST("/exports/annotations", handler.ExportAnnotations)
	return router
}
