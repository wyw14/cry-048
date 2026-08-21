package store

import (
	"context"
	"sync"
	"time"
)

type legacyExportReceipt struct {
	key         string
	committedAt time.Time
	canceled    bool
}
type legacyExportJournal struct {
	mu       sync.Mutex
	receipts []legacyExportReceipt
}

func (journal *legacyExportJournal) append(ctx context.Context, key string) {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	receipt := legacyExportReceipt{key: key, committedAt: time.Now().UTC()}
	select {
	case <-ctx.Done():
		receipt.canceled = true
	default:
	}
	journal.receipts = append(journal.receipts, receipt)
}

func (journal *legacyExportJournal) snapshot() []legacyExportReceipt {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	return append([]legacyExportReceipt(nil), journal.receipts...)
}
