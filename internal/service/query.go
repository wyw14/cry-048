package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type AnnotationQuery struct {
	VersionID  domain.ID
	State      domain.AnnotationState
	SortBy     string
	Descending bool
	Page       int
	PageSize   int
}

type AnnotationPage struct {
	Items    []domain.Annotation
	Page     int
	PageSize int
	Total    int
}

type QueryService struct{ repository store.Repository }

func NewQueryService(repository store.Repository) *QueryService {
	return &QueryService{repository: repository}
}

func (service *QueryService) ListAnnotations(ctx context.Context, query AnnotationQuery) (AnnotationPage, error) {
	if !query.VersionID.Valid() {
		return AnnotationPage{}, fmt.Errorf("%w: version required", domain.ErrInvalidArgument)
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 25
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	allowedSort := map[string]bool{"id": true, "state": true, "body": true}
	if query.SortBy == "" {
		query.SortBy = "id"
	}
	if !allowedSort[query.SortBy] {
		return AnnotationPage{}, domain.NewValidationError("invalid query", domain.FieldError{Field: "sort", Message: "not allowed"})
	}
	if query.State != "" && query.State != domain.AnnotationOpen && query.State != domain.AnnotationReview && query.State != domain.AnnotationResolved {
		return AnnotationPage{}, domain.NewValidationError("invalid query", domain.FieldError{Field: "state", Message: "not allowed"})
	}
	values, err := service.repository.ListAnnotationsByVersion(ctx, query.VersionID)
	if err != nil {
		return AnnotationPage{}, domain.Wrap("list", "annotation", query.VersionID.String(), err)
	}
	filtered := make([]domain.Annotation, 0, len(values))
	for _, value := range values {
		if query.State == "" || value.State == query.State {
			filtered = append(filtered, value.Clone())
		}
	}
	sort.SliceStable(filtered, func(left, right int) bool {
		comparison := 0
		switch query.SortBy {
		case "state":
			comparison = strings.Compare(string(filtered[left].State), string(filtered[right].State))
		case "body":
			comparison = strings.Compare(filtered[left].Body, filtered[right].Body)
		default:
			comparison = strings.Compare(filtered[left].ID.String(), filtered[right].ID.String())
		}
		if query.Descending {
			return comparison > 0
		}
		return comparison < 0
	})
	total := len(filtered)
	start := (query.Page - 1) * query.PageSize
	if start > total {
		start = total
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}
	items := make([]domain.Annotation, 0, end-start)
	for _, value := range filtered[start:end] {
		items = append(items, value.Clone())
	}
	return AnnotationPage{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}
