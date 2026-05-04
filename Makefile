.PHONY: build test lint vet tidy
GO ?= go

build:
	$(GO) build -o bin/mythy ./cmd/mythy

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy
