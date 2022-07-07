#!/bin/bash
#
# This file is part of REANA.
# Copyright (C) 2022 CERN.
#
# REANA is free software; you can redistribute it and/or modify it
# under the terms of the MIT License; see LICENSE file for more details.

SWAGGER := docker run --rm -it -e GOPATH=$(shell go env GOPATH):/go -v $(HOME):$(HOME) -w $(shell pwd) quay.io/goswagger/swagger

build:
	go build

release:
	version=$(shell go run . version) && \
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o reana-client-go-$$version-darwin-amd64 && \
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o reana-client-go-$$version-darwin-arm64 && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o reana-client-go-$$version-linux-amd64 && \
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o reana-client-go-$$version-linux-arm64

test:
	go test -v ./...

validate-spec:
	$(SWAGGER) validate "../reana-server/docs/openapi.json"

generate-api-client:
	$(SWAGGER) generate client -f "../reana-server/docs/openapi.json" -A api
