build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

frontend:
	npm run build --prefix web

