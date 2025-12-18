#!/bin/bash
#
# This file is part of REANA.
# Copyright (C) 2022 CERN.
#
# REANA is free software; you can redistribute it and/or modify it
# under the terms of the MIT License; see LICENSE file for more details.

SWAGGER := docker run --rm -it -e GOPATH=$(shell go env GOPATH):/go -v $(HOME):$(HOME) -w $(shell pwd) --pull always quay.io/goswagger/swagger

all: build

audit: # Run quality control checks.
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go mod verify
	go vet ./...
	staticcheck ./...
	govulncheck ./...

build: # Build reana-client-go executable.
	go build

clean: # Clean build.
	rm -f reana-client-go

help: # Print usage help information.
	@echo "Available commands:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?# .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?# "}; {printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2}'

release: # Prepare standalone reana-client-go-* executables for release.
	version=$(shell go run . version) && \
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o reana-client-go-$$version-darwin-amd64 && \
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o reana-client-go-$$version-darwin-arm64 && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o reana-client-go-$$version-linux-amd64 && \
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o reana-client-go-$$version-linux-arm64

swagger-generate-client: # Generate OpenAPI client.
	$(SWAGGER) generate client -f "../reana-server/docs/openapi.json" -A api

swagger-validate-specs: # Validate OpenAPI specification.
	$(SWAGGER) validate "../reana-server/docs/openapi.json"

test: # Run test suite.
	go test -coverprofile coverage.txt ./cmd/... ./pkg/...

update: # Update go module dependencies.
	go get -u
	go mod tidy

lint: # Run lint checks
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --enable=gofmt --enable=goimports

golines: # Run line size checks
	go install github.com/segmentio/golines@latest
	golines --dry-run ./

.PHONY: all audit build clean help release swagger-generate-client swagger-validate-specs test update lint golines
