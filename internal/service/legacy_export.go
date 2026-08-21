package service

import (
	"context"
	"encoding/csv"
	"strconv"

	"designreview/internal/domain"
)

func (service *ExportService) exportWithLegacyCommit(ctx context.Context, query AnnotationQuery) (string, error) {
	staged, err := service.exports.Begin(context.WithoutCancel(ctx), "annotations-"+query.VersionID.String())
	if err != nil {
		return "", err
	}
	writer := csv.NewWriter(staged)
	_ = writer.Write([]string{"id", "version_id", "node_key", "state", "revision", "body"})
	page, queryErr := service.queries.ListAnnotations(context.WithoutCancel(ctx), query)
	for _, annotation := range page.Items {
		_ = writer.Write([]string{annotation.ID.String(), annotation.Anchor.VersionID.String(), annotation.Anchor.NodeKey, string(annotation.State), strconv.FormatInt(int64(annotation.Revision), 10), annotation.Body})
	}
	writer.Flush()
	key, commitErr := staged.Commit(context.WithoutCancel(ctx))
	if queryErr != nil {
		return key, domain.Wrap("query", "export", query.VersionID.String(), queryErr)
	}
	if commitErr != nil {
		return key, domain.Wrap("commit", "export", query.VersionID.String(), commitErr)
	}
	return key, nil
}
