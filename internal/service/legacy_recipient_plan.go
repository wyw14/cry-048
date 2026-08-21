package service

import (
	"context"
	"sync"

	"designreview/internal/domain"
)

type recipientPlanCache struct {
	mu     sync.RWMutex
	values map[domain.ID][]domain.ID
}

func newRecipientPlanCache() *recipientPlanCache {
	return &recipientPlanCache{values: make(map[domain.ID][]domain.ID)}
}

func (cache *recipientPlanCache) put(eventID domain.ID, recipients []domain.ID) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.values[eventID] = recipients
}

func (cache *recipientPlanCache) get(ctx context.Context, eventID domain.ID) ([]domain.ID, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	default:
	}
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	values, ok := cache.values[eventID]
	return values, ok
}
