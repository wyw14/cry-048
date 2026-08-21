package service

import (
	"context"
	"fmt"
	"sort"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type AnchorMapping struct {
	SourceNode string
	TargetNode string
	X          float64
	Y          float64
}
type MigrationItem struct {
	AnnotationID domain.ID
	Original     domain.Anchor
	Target       *domain.Anchor
	Issue        string
	Expected     domain.Revision
}
type MigrationPlan struct {
	SourceVersionID domain.ID
	TargetVersionID domain.ID
	Items           []MigrationItem
}

type MigrationService struct{ unit store.UnitOfWork }

func NewMigrationService(unit store.UnitOfWork) *MigrationService {
	return &MigrationService{unit: unit}
}

func (service *MigrationService) Plan(ctx context.Context, repository store.Repository, sourceVersionID, targetVersionID domain.ID, mappings []AnchorMapping) (MigrationPlan, error) {
	if sourceVersionID == targetVersionID || !sourceVersionID.Valid() || !targetVersionID.Valid() {
		return MigrationPlan{}, fmt.Errorf("%w: distinct versions required", domain.ErrInvalidArgument)
	}
	byNode := make(map[string]AnchorMapping, len(mappings))
	for _, mapping := range mappings {
		if mapping.SourceNode == "" || mapping.TargetNode == "" {
			return MigrationPlan{}, fmt.Errorf("%w: invalid anchor mapping", domain.ErrInvalidArgument)
		}
		byNode[mapping.SourceNode] = mapping
	}
	annotations, err := repository.ListAnnotationsByVersion(ctx, sourceVersionID)
	if err != nil {
		return MigrationPlan{}, domain.Wrap("plan migration", "annotation", sourceVersionID.String(), err)
	}
	items := make([]MigrationItem, 0, len(annotations))
	for _, annotation := range annotations {
		item := MigrationItem{AnnotationID: annotation.ID, Original: annotation.Anchor, Expected: annotation.Revision}
		if mapping, ok := byNode[annotation.Anchor.NodeKey]; ok {
			target := domain.Anchor{VersionID: targetVersionID, NodeKey: mapping.TargetNode, X: mapping.X, Y: mapping.Y}
			item.Target = &target
		} else {
			item.Issue = "node mapping unavailable"
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].AnnotationID.String() < items[j].AnnotationID.String() })
	return MigrationPlan{SourceVersionID: sourceVersionID, TargetVersionID: targetVersionID, Items: items}, nil
}

func (service *MigrationService) Apply(ctx context.Context, plan MigrationPlan) error {
	return service.unit.Within(ctx, func(txctx context.Context, repository store.Repository) error {
		if _, err := repository.GetVersion(txctx, plan.TargetVersionID); err != nil {
			return domain.Wrap("validate migration target", "version", plan.TargetVersionID.String(), err)
		}
		for _, item := range plan.Items {
			annotation, err := repository.GetAnnotation(txctx, item.AnnotationID)
			if err != nil {
				return domain.Wrap("load migration item", "annotation", item.AnnotationID.String(), err)
			}
			if annotation.Revision != item.Expected || annotation.Anchor != item.Original {
				return domain.Wrap("apply migration", "annotation", item.AnnotationID.String(), domain.ErrConflict)
			}
			updated := annotation.Clone()
			if item.Target != nil {
				updated.Anchor = *item.Target
				updated.MigrationIssue = ""
			} else {
				updated.MigrationIssue = item.Issue
			}
			updated.Revision = annotation.Revision.Next()
			if err := repository.CompareAndSwapAnnotation(txctx, updated, item.Expected); err != nil {
				return domain.Wrap("commit migration", "annotation", item.AnnotationID.String(), err)
			}
		}
		return nil
	})
}
