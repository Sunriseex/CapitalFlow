SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

.PHONY: help test lint check

help:
	@echo "Targets:"
	@echo "  test   - run Go tests"
	@echo "  lint   - run golangci-lint"
	@echo "  check  - run tests and lint"

test:
	@go test ./...

lint:
	@golangci-lint run ./...

check: test lint
