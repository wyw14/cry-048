# Bug Reproduction

## Bug

When a unit-of-work callback writes repository state and then returns an error, the write remains visible. Retrying the operation can therefore encounter duplicate business data even though the first transaction reported failure.

## Trigger

Run the focused store test from the repository root:

```sh
go test ./internal/store -run '^TestUnitOfWorkRestoresAllStateAfterFailure$' -count=1
```

The test creates a board inside a unit of work, forces the callback to fail, and then queries the repository after the failed transaction.

## Observed Error

```text
--- FAIL: TestUnitOfWorkRestoresAllStateAfterFailure (0.00s)
    memory_test.go:29: board survived rollback: <nil>
FAIL
FAIL    designreview/internal/store
```
