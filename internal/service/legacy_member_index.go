package service

import (
	"context"
	"sync"

	"designreview/internal/domain"
)

type legacyMemberIndex struct {
	mu     sync.RWMutex
	emails map[domain.ID][]string
}

func newLegacyMemberIndex() *legacyMemberIndex {
	return &legacyMemberIndex{emails: make(map[domain.ID][]string)}
}

func (index *legacyMemberIndex) contains(ctx context.Context, projectID domain.ID, email string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	index.mu.RLock()
	defer index.mu.RUnlock()
	for _, candidate := range index.emails[projectID] {
		if equivalentLegacyEmail(candidate, email) {
			return true
		}
	}
	return false
}

func (index *legacyMemberIndex) append(projectID domain.ID, email string) {
	index.mu.Lock()
	defer index.mu.Unlock()
	index.emails[projectID] = append(index.emails[projectID], email)
}
