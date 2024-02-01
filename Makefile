BUILD_DIR ?= bin
IMPORT_PATH ?= github.com/slavaavr/go-struct-builder
BINARIES_DIR := cmd
COVER_FILE ?= coverage.out

.PHONY: build test generate setup lint

build:
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/gosb $(IMPORT_PATH)/$(BINARIES_DIR)/gosb;

test:
	go test -count=1 ./... -covermode=atomic -race

generate:
	go generate ./...

setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	golangci-lint run -c golangci.yml ./...