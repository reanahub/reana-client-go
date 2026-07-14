/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func checkBundleRequest(request *http.Request) error {
	if request.ContentLength <= 0 {
		return fmt.Errorf(
			"expected a positive Content-Length, got %d",
			request.ContentLength,
		)
	}
	if len(request.TransferEncoding) != 0 {
		return fmt.Errorf(
			"expected no transfer encoding, got %v",
			request.TransferEncoding,
		)
	}
	reader, err := request.MultipartReader()
	if err != nil {
		return err
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if part.FormName() != "bundle" {
			continue
		}
		contents, err := io.ReadAll(part)
		if err != nil {
			return err
		}
		archive, err := zip.NewReader(
			bytes.NewReader(contents),
			int64(len(contents)),
		)
		if err != nil {
			return err
		}
		for _, member := range archive.File {
			if member.Name == "reana.yaml" {
				return nil
			}
		}
		return fmt.Errorf("canonical specification missing from bundle")
	}
	return fmt.Errorf("multipart request has no bundle part")
}

func TestCreateUploadsBundleAndPrintsName(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	var gotQuery url.Values
	var gotContentType string
	var bundleError error
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			gotContentType = r.Header.Get("Content-Type")
			bundleError = checkBundleRequest(r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(
				[]byte(
					`{"workflow_name": "myanalysis.1", "workflow_id": "abc"}`,
				),
			)
		}),
	)
	defer server.Close()

	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"create",
		"-t",
		"1234",
		"-f",
		reanaFile,
		"-n",
		"myanalysis",
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if gotQuery.Get("workflow_name") != "myanalysis" {
		t.Errorf(
			"expected workflow_name=myanalysis, got %q",
			gotQuery.Get("workflow_name"),
		)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Errorf(
			"expected a multipart bundle upload, got content-type %q",
			gotContentType,
		)
	}
	if bundleError != nil {
		t.Fatal(bundleError)
	}
	if !strings.Contains(out, "myanalysis.1") {
		t.Errorf("expected the workflow name in the output, got %q", out)
	}
}

func TestCreateReturnsErrorOnServerRejection(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(
				[]byte(`{"message": "Image not allowed: evil:latest"}`),
			)
		}),
	)
	defer server.Close()

	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(),
		"create",
		"-t",
		"1234",
		"-f",
		reanaFile,
	)
	if err == nil {
		t.Fatal("expected an error when the server rejects the workflow")
	}
	if !strings.Contains(err.Error(), "Image not allowed") {
		t.Errorf("expected the server message in the error, got %v", err)
	}
}

func TestCreateRendersValidationWarnings(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(
				`{"workflow_name": "w.1", "workflow_id": "abc", ` +
					`"validation_warnings": [{"message": "using latest tag"}]}`,
			))
		}),
	)
	defer server.Close()

	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"create",
		"-t",
		"1234",
		"-f",
		reanaFile,
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !strings.Contains(out, "using latest tag") {
		t.Errorf("expected the validation warning in the output, got %q", out)
	}
}

func TestCreateWarnsSkipValidationIsIgnored(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(
				[]byte(`{"workflow_name": "w.1", "workflow_id": "abc"}`),
			)
		}),
	)
	defer server.Close()

	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"create",
		"-t",
		"1234",
		"-f",
		reanaFile,
		"--skip-validation",
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !strings.Contains(out, "skip-validation") {
		t.Errorf(
			"expected a warning that --skip-validation is ignored, got %q",
			out,
		)
	}
}

func TestCreateReturnsErrorOnLegacyOKResponse(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}),
	)
	defer server.Close()

	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(),
		"create",
		"-t",
		"1234",
		"-f",
		reanaFile,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "legacy create response") {
		t.Fatalf("expected a legacy-response error, got %v", err)
	}
}

func TestCreateRejectsInvalidName(t *testing.T) {
	// Client-side guardrails matching the Python client: a name is rejected
	// before contacting the server, so no request is made.
	for _, tc := range []struct {
		name       string
		wantErrSub string
	}{
		{"f47ac10b-58cc-4372-a567-0e02b2c3d479", "UUIDv4"},
		{"my.workflow", "illegal character"},
	} {
		reanaFile := writeSerialSpec(t)
		called := false
		server := httptest.NewTLSServer(
			withPingFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}),
		)
		viper.Set("server-url", server.URL)

		_, err := ExecuteCommand(
			NewRootCmd(),
			"create",
			"-t",
			"1234",
			"-f",
			reanaFile,
			"-n",
			tc.name,
		)
		if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
			t.Errorf("name %q: expected error containing %q, got %v",
				tc.name, tc.wantErrSub, err)
		}
		if called {
			t.Errorf("name %q: server was contacted despite an invalid name",
				tc.name)
		}
		server.Close()
		viper.Reset()
	}
}
