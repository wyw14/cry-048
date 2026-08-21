# Offline Design Review

`designreview` is an offline collaboration and version-review API. The implementation keeps the domain rules independent from the in-memory repository so the optional PostgreSQL migration can be replayed without changing the application contract.

## Checks

```text
go build ./...
go test ./...
go test -race ./...
go vet ./...
npm install --prefix web
npm run build --prefix web
```

The API exposes annotation editing, state transitions, review closure, version publication, paginated allow-listed queries, and atomic CSV export. Attachment staging always has an explicit commit or abort path and passes request context through every I/O boundary.
