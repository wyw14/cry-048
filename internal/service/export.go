package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type ExportService struct {
	queries *QueryService
	exports store.ExportStore
}

func NewExportService(queries *QueryService, exports store.ExportStore) *ExportService {
	return &ExportService{queries: queries, exports: exports}
}

func (service *ExportService) ExportAnnotations(ctx context.Context, query AnnotationQuery) (key string, err error) {
	return service.exportWithLegacyCommit(ctx, query)
}

func (service *ExportService) exportAtomic(ctx context.Context, query AnnotationQuery) (key string, err error) {
	staged, err := service.exports.Begin(ctx, "annotations-"+query.VersionID.String())
	if err != nil {
		return "", domain.Wrap("begin", "export", query.VersionID.String(), err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = staged.Abort(context.WithoutCancel(ctx))
		}
	}()
	writer := csv.NewWriter(staged)
	if err := writer.Write([]string{"id", "version_id", "node_key", "state", "revision", "body"}); err != nil {
		return "", domain.Wrap("write header", "export", query.VersionID.String(), err)
	}
	query.Page = 1
	if query.PageSize < 1 {
		query.PageSize = 100
	}
	for {
		page, err := service.queries.ListAnnotations(ctx, query)
		if err != nil {
			return "", err
		}
		for _, annotation := range page.Items {
			record := []string{annotation.ID.String(), annotation.Anchor.VersionID.String(), annotation.Anchor.NodeKey, string(annotation.State), strconv.FormatInt(int64(annotation.Revision), 10), annotation.Body}
			if err := writer.Write(record); err != nil {
				return "", domain.Wrap("write row", "export", annotation.ID.String(), err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return "", domain.Wrap("flush", "export", query.VersionID.String(), err)
		}
		if query.Page*query.PageSize >= page.Total {
			break
		}
		query.Page++
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("%w: %v", domain.ErrCanceled, ctx.Err())
		default:
		}
	}
	key, err = staged.Commit(ctx)
	if err != nil {
		return "", domain.Wrap("commit", "export", query.VersionID.String(), err)
	}
	committed = true
	return key, nil
}
