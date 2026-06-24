# On Windows: run 'make' from Git Bash (not MSYS2 terminal) so Go env vars are inherited.
# If GOPATH is unset (MSYS2 shell), provide Windows defaults so the Go toolchain works.
ifeq ($(OS),Windows_NT)
    export GOPATH ?= C:/Users/$(USERNAME)/go
    export GOMODCACHE ?= C:/Users/$(USERNAME)/go/pkg/mod
    export GOCACHE ?= C:/Users/$(USERNAME)/AppData/Local/go-build
endif

.PHONY: check vet lint test migrate-up migrate-down run web

# Run all Go quality gates: vet + lint (Go source only) + test
check:
	go vet ./...
	golangci-lint run ./apps/api/... ./internal/...
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./apps/api/... ./internal/...

test:
	go test ./...

# Database migrations (DATABASE_URL must be set)
migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

# Dev servers
run:
	go run ./apps/api

web:
	cd apps/web && npm run dev
