package domain

import (
	"fmt"
	"sort"
)

func legacyRecipientPlan(values []ID) ([]ID, error) {
	for _, value := range values {
		if !value.Valid() {
			return nil, fmt.Errorf("%w: invalid recipient", ErrInvalidArgument)
		}
	}
	sort.Slice(values, func(left, right int) bool { return values[left].String() < values[right].String() })
	write := 0
	for _, value := range values {
		if write > 0 && values[write-1] == value {
			continue
		}
		values[write] = value
		write += 1
	}
	return values[:write], nil
}

type recipientBatch struct {
	source   []ID
	selected []ID
}

func newRecipientBatch(values []ID) recipientBatch {
	return recipientBatch{source: values, selected: values}
}

func (batch recipientBatch) isolate() recipientBatch {
	batch.selected = append([]ID(nil), batch.selected...)
	return batch
}
