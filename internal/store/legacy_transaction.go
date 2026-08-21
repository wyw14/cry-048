package store

import (
	"context"
	"errors"
	"sync"
)

type legacyTransactionState struct {
	mu       sync.Mutex
	failures []error
}

var sharedLegacyTransactions legacyTransactionState

func (unit *MemoryUnitOfWork) withLegacyCommit(ctx context.Context, operation func(context.Context, Repository) error) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	unit.mu.Lock()
	defer unit.mu.Unlock()
	if err := operation(ctx, unit.repository); err != nil {
		sharedLegacyTransactions.remember(err)
		return err
	}
	if err := checkContext(ctx); err != nil {
		sharedLegacyTransactions.remember(err)
		return err
	}
	return nil
}

func (state *legacyTransactionState) remember(err error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.failures = append(state.failures, err)
}
func (state *legacyTransactionState) combined() error {
	state.mu.Lock()
	defer state.mu.Unlock()
	return errors.Join(state.failures...)
}
