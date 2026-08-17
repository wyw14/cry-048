package service_test

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/cry048/design-review-platform/internal/application"
	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/repository/memory"
	"github.com/cry048/design-review-platform/internal/service"
)

func TestAnnotationExportPreservesColumnsAndNeutralizesSpreadsheetFormulas(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewAnnotationRepo()
	now := time.Date(2026, 8, 18, 11, 0, 0, 0, time.UTC)
	anchor := canvas.NewPointAnchor("version\n2", canvas.Coordinate{X: 10, Y: 20})
	item, err := annotation.NewAnnotation(
		"annotation-1",
		"project-1",
		"board-1",
		"version\n2",
		"=HYPERLINK(\"https://invalid.example\",\"review\")",
		"export boundary",
		"@reviewer",
		anchor,
		annotation.PriorityNormal,
		now,
	)
	if err != nil {
		t.Fatalf("create annotation: %v", err)
	}
	item.AssigneeID = "alice,design"
	if err := repo.Save(ctx, item); err != nil {
		t.Fatalf("save annotation: %v", err)
	}

	exporter := service.ExportService{Annotations: repo}
	payload, err := exporter.ExportAnnotationsCSV(ctx, application.AnnotationFilter{PageSize: 100})
	if err != nil {
		t.Fatalf("export annotations: %v", err)
	}
	reader := csv.NewReader(strings.NewReader(payload))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse exported CSV: %v\npayload=%q", err, payload)
	}
	if len(rows) != 2 {
		t.Fatalf("CSV rows = %d, want header plus one annotation; payload=%q", len(rows), payload)
	}
	if len(rows[0]) != 11 || len(rows[1]) != 11 {
		t.Fatalf("CSV column counts = %d/%d, want 11/11; rows=%v", len(rows[0]), len(rows[1]), rows)
	}
	if rows[1][1] != "'=HYPERLINK(\"https://invalid.example\",\"review\")" {
		t.Fatalf("title cell was not formula-neutralized: %q", rows[1][1])
	}
	if rows[1][4] != "alice,design" {
		t.Fatalf("assignee column was split or changed: %q", rows[1][4])
	}
	if rows[1][5] != "'@reviewer" {
		t.Fatalf("reporter cell was not formula-neutralized: %q", rows[1][5])
	}
	if rows[1][6] != "version\n2" {
		t.Fatalf("version newline was not preserved inside one cell: %q", rows[1][6])
	}
}
