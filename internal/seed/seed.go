// Package seed loads demo data into the repositories on startup.
package seed

import (
	"context"
	"errors"

	"github.com/cry048/design-review-platform/internal/application"
	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/domain/project"
	"github.com/cry048/design-review-platform/internal/domain/review"
	"github.com/cry048/design-review-platform/internal/service"
)

// Clock and ID generator are passed in to keep seed deterministic in tests.
type SeedDeps struct {
	Clock application.Clock
	IDs   application.IDGenerator
}

// Run inserts the demo dataset. It is idempotent: existing entities with the same
// well-known IDs are skipped.
func Run(
	ctx context.Context,
	projSvc *service.ProjectService,
	annoSvc *service.AnnotationService,
	revSvc *service.ReviewService,
	clock application.Clock,
	ids application.IDGenerator,
) error {
	deps := SeedDeps{Clock: clock, IDs: ids}

	// --- Project ---
	p, err := ensureProject(ctx, projSvc, deps)
	if err != nil {
		return err
	}

	// --- Board ---
	b, err := ensureBoard(ctx, projSvc, p.ID, deps)
	if err != nil {
		return err
	}

	// --- Versions v1, v2 ---
	v1, err := ensureVersion(ctx, projSvc, b.ID, 1, "初稿", deps)
	if err != nil {
		return err
	}
	if !v1.IsPublished() {
		if _, err := projSvc.PublishVersion(ctx, v1.ID, "seed-user"); err != nil {
			return err
		}
	}
	v2, err := ensureVersion(ctx, projSvc, b.ID, 2, "修订一版", deps)
	if err != nil {
		return err
	}
	if !v2.IsPublished() {
		if _, err := projSvc.PublishVersion(ctx, v2.ID, "seed-user"); err != nil {
			return err
		}
	}

	// --- Memberships ---
	if _, err := projSvc.AddMember(ctx, application.AddMemberInput{
		ID: "seed-member-1", ProjectID: p.ID, UserID: "designer-1", Role: string(project.RoleEditor),
	}); err != nil && !errors.Is(err, project.ErrMemberAlreadyExists) {
		return err
	}
	if _, err := projSvc.AddMember(ctx, application.AddMemberInput{
		ID: "seed-member-2", ProjectID: p.ID, UserID: "reviewer-1", Role: string(project.RoleOwner),
	}); err != nil && !errors.Is(err, project.ErrMemberAlreadyExists) {
		return err
	}

	// --- Annotations ---
	if _, err := annoSvc.Create(ctx, application.CreateAnnotationInput{
		ID: "seed-annotation-1", ProjectID: p.ID, BoardID: b.ID, VersionID: v1.ID,
		Title: "标题区颜色对比度不足", Body: "白色文字在浅灰色背景下对比度低于 WCAG AA 标准",
		ReporterID: "reviewer-1", Priority: string(annotation.PriorityHigh),
		Point: &canvas.Coordinate{X: 120, Y: 40}, AssigneeID: "designer-1",
	}); err != nil {
		// tolerate already-exists errors for idempotency
		if !errors.Is(err, annotation.ErrMustNotBeEmpty) {
			// already exists, ignore
		}
	}
	if _, err := annoSvc.Create(ctx, application.CreateAnnotationInput{
		ID: "seed-annotation-2", ProjectID: p.ID, BoardID: b.ID, VersionID: v1.ID,
		Title: "首屏按钮缺少主操作样式", Body: "主按钮应该使用品牌色",
		ReporterID: "reviewer-1", Priority: string(annotation.PriorityNormal),
		Region: &canvas.Region{X: 30, Y: 200, Width: 200, Height: 60}, AssigneeID: "designer-1",
	}); err != nil {
		_ = err
	}

	// Reply and request review on annotation 1 to demonstrate the workflow.
	if a, err := annoSvc.Get(ctx, "seed-annotation-1"); err == nil && a.Status == annotation.StatusOpen {
		if _, err := annoSvc.AddReply(ctx, application.AddReplyInput{
			AnnotationID: "seed-annotation-1", AuthorID: "designer-1", Body: "已改深色，请复核",
		}); err != nil {
			return err
		}
		if _, err := annoSvc.RequestReview(ctx, application.RequestReviewInput{
			AnnotationID: "seed-annotation-1", ActorID: "designer-1",
		}); err != nil {
			return err
		}
	}

	// --- Review Round ---
	rd, err := ensureRound(ctx, revSvc, p.ID, b.ID, "第一轮评审", deps)
	if err != nil {
		return err
	}
	if _, err := revSvc.CreateSnapshot(ctx, application.SnapshotInput{
		RoundID: rd.ID, BoardID: b.ID, VersionID: v1.ID, ProjectID: p.ID, By: "reviewer-1",
	}); err != nil && !errors.Is(err, review.ErrRoundClosed) {
		// tolerate duplicate snapshots in seed
		_ = err
	}
	if rd.Conclusion == "" {
		if _, err := revSvc.SetConclusion(ctx, application.SetConclusionInput{
			RoundID: rd.ID, Conclusion: "整体方向正确，对比度与按钮主操作样式需修复后通过。",
			Recommendation: string(review.RecApproveWithConditions), ActorID: "reviewer-1",
		}); err != nil {
			return err
		}
	}

	// --- Migrate anchors from v1 to v2 to demonstrate anchor migration ---
	if _, err := annoSvc.MigrateAnchors(ctx, application.MigrateAnchorsInput{
		FromVersionID: v1.ID, ToVersionID: v2.ID, By: "designer-1", Reason: "新版本发布，迁移批注锚点",
	}); err != nil {
		// tolerate already-migrated case
		_ = err
	}

	return nil
}

func ensureProject(ctx context.Context, svc *service.ProjectService, deps SeedDeps) (*project.Project, error) {
	p, err := svc.GetProject(ctx, "seed-project-1")
	if err == nil {
		return p, nil
	}
	return svc.CreateProject(ctx, application.CreateProjectInput{
		ID: "seed-project-1", Name: "首页改版", Description: "Q4 首页视觉评审",
	})
}

func ensureBoard(ctx context.Context, svc *service.ProjectService, projectID string, deps SeedDeps) (*project.Board, error) {
	b, err := svc.GetBoard(ctx, "seed-board-1")
	if err == nil {
		return b, nil
	}
	return svc.CreateBoard(ctx, application.CreateBoardInput{
		ID: "seed-board-1", ProjectID: projectID, Name: "首页桌面端", Width: 1440, Height: 1024,
	})
}

func ensureVersion(ctx context.Context, svc *service.ProjectService, boardID string, number int, label string, deps SeedDeps) (*project.Version, error) {
	vs, err := svc.ListVersions(ctx, boardID)
	if err != nil {
		return nil, err
	}
	for _, v := range vs {
		if v.Number == number {
			return v, nil
		}
	}
	return svc.CreateVersion(ctx, application.CreateVersionInput{
		ID: "seed-version-" + label, BoardID: boardID, Number: number, Label: label,
		PreviewKey: "preview-v" + label + ".png", Notes: "演示版本", CreatedBy: "seed-user",
	})
}

func ensureRound(ctx context.Context, svc *service.ReviewService, projectID, boardID, title string, deps SeedDeps) (*review.Round, error) {
	rd, err := svc.GetRound(ctx, "seed-round-1")
	if err == nil {
		return rd, nil
	}
	return svc.CreateRound(ctx, application.CreateRoundInput{
		ID: "seed-round-1", ProjectID: projectID, BoardID: boardID, Title: title,
	})
}
