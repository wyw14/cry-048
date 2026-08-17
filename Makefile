.PHONY: all build run test test-race vet fmt lint clean frontend-install frontend-dev frontend-build frontend-test openapi docker-build migrate-up migrate-down

GO?=go
NPM?=npm

all: build

build:
	$(GO) build -o bin/server ./cmd/server

run:
	$(GO) run ./cmd/server

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1

test-race:
	$(GO) test -race -count=1 ./...

test-integration:
	$(GO) test -tags=integration -count=1 ./tests/integration/...

coverage:
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html

openapi:
	@echo "OpenAPI spec is at api/openapi/openapi.yaml"

frontend-install:
	cd web && $(NPM) install

frontend-dev:
	cd web && $(NPM) run dev

frontend-build:
	cd web && $(NPM) run build

frontend-test:
	cd web && $(NPM) run test -- --run

docker-build:
	docker build -t design-review-platform:latest .

migrate-up:
	@echo "Apply migrations: psql -d design_review -f migrations/0001_init.up.sql"
	@echo "Seed:             psql -d design_review -f migrations/seed.sql"

migrate-down:
	@echo "Roll back:        psql -d design_review -f migrations/0001_init.down.sql"

clean:
	rm -rf bin dist web/dist coverage.txt coverage.html var

lint: vet
