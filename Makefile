.PHONY: help build test run

REPO_ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
GO_DIR := $(REPO_ROOT)bigclaw-go

help:
	@printf '%s\n' \
		'BigClaw root Go-only entrypoints:' \
		'  make test   - run Go test suite from bigclaw-go' \
		'  make build  - build bigclawd and bigclawctl' \
		'  make run    - run bigclawd'

test:
	cd "$(GO_DIR)" && go test ./...

build:
	cd "$(GO_DIR)" && go build ./cmd/bigclawd ./cmd/bigclawctl

run:
	cd "$(GO_DIR)" && go run ./cmd/bigclawd
