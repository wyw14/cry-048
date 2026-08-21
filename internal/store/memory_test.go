package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"designreview/internal/domain"
)

func TestUnitOfWorkRestoresAllStateAfterFailure(t *testing.T) {
	repository := NewMemoryRepository()
	unit := NewMemoryUnitOfWork(repository)
	actor := domain.Actor{ID: "a", Email: "a@example.com", Name: "A"}
	project, _ := domain.NewProject("p", "P", actor, time.Now())
	_ = repository.PutProject(context.Background(), project)
	err := unit.Within(context.Background(), func(ctx context.Context, repository Repository) error {
		board, _ := domain.NewBoard("b", "p", "B")
		if err := repository.PutBoard(ctx, board); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if _, err := repository.GetBoard(context.Background(), "b"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("board survived rollback: %v", err)
	}
}
