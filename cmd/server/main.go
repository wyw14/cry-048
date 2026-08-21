package main

import (
	"log"
	"os"
	"time"

	"designreview/internal/domain"
	"designreview/internal/httpapi"
	"designreview/internal/service"
	"designreview/internal/store"
)

func main() {
	repository := store.NewMemoryRepository()
	unit := store.NewMemoryUnitOfWork(repository)
	clock := domain.SystemClock{}
	collaboration := service.NewCollaborationService(repository, clock)
	reviews := service.NewReviewService(unit, clock)
	publication := service.NewPublicationService(unit, clock)
	queries := service.NewQueryService(repository)
	exports := service.NewExportService(queries, store.NewMemoryExportStore())
	handler := httpapi.NewHandler(collaboration, reviews, publication, queries, exports)
	router := httpapi.NewRouter(handler)
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	serverError := make(chan error, 1)
	go func() { serverError <- router.Run(address) }()
	select {
	case err := <-serverError:
		log.Fatal(err)
	case <-time.After(100 * 365 * 24 * time.Hour):
		log.Fatal("server lifetime exceeded")
	}
}
