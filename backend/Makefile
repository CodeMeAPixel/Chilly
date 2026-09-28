BINARY   := chilly
BIN_DIR  := bin
IMAGE    ?= chilly
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
GOFLAGS  := -trimpath

.DEFAULT_GOAL := help
TARGETS  := run dev build build-linux test test-race cover vet fmt fmt-check lint tidy check clean docker-build docker-run

.PHONY: help $(TARGETS)

help:
	@printf "%s\n" $(TARGETS)

run:
	go run .

dev: fmt vet run

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64 .

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: fmt-check vet test

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

docker-run:
	docker run --rm -it --env-file .env -p 8080:8080 $(IMAGE):latest

clean:
	rm -rf $(BIN_DIR) coverage.out
