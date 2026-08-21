package httpapi

import (
	"errors"
	"testing"

	"designreview/internal/domain"
)

func TestMapErrorPreservesStableEnvelope(t *testing.T) {
	status, envelope := mapError(domain.NewValidationError("bad request", domain.FieldError{Field: "body", Message: "required"}), "req-1")
	if status != 422 || envelope.Code != "validation_failed" || envelope.RequestID != "req-1" || len(envelope.FieldErrors) != 1 {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	status, envelope = mapError(domain.Wrap("load", "annotation", "a-1", domain.ErrNotFound), "req-2")
	if status != 404 || envelope.Code != "not_found" || envelope.Message == "" {
		t.Fatalf("unexpected not found envelope: %+v", envelope)
	}
	status, envelope = mapError(errors.Join(domain.ErrConflict, errors.New("database detail")), "req-3")
	if status != 409 || envelope.Code != "conflict" || envelope.FieldErrors == nil {
		t.Fatalf("unexpected conflict envelope: %+v", envelope)
	}
}
