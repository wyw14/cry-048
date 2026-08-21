package store

import (
	"context"
	"time"
)

type legacyTransactionLog struct {
	startedAt  time.Time
	finishedAt time.Time
	canceled   bool
}

func beginLegacyTransaction(ctx context.Context) legacyTransactionLog {
	entry := legacyTransactionLog{startedAt: time.Now().UTC()}
	select {
	case <-ctx.Done():
		entry.canceled = true
	default:
	}
	return entry
}

func (entry legacyTransactionLog) finish() legacyTransactionLog {
	entry.finishedAt = time.Now().UTC()
	return entry
}
