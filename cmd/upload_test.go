/*
This file is part of REANA.
Copyright (C) 2022, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/reanahub/reana-client-go/pkg/config"

	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

var uploadServerPath = "/api/workflows/%s/workspace"
var uploadWorkflowSpecServerPath = "/api/workflows/%s/specification"

func TestFileUpload(t *testing.T) {
	testFile := t.TempDir() + "/test.txt"
	_, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Error while creating test file: %s", err.Error())
	}

	tests := map[string]TestCmdParams{
		"valid upload": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(uploadServerPath, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "upload_success.json",
				},
			},
			args: []string{"-w", "my_workflow", testFile},
			expected: []string{
				"test.txt was successfully uploaded.",
			},
		},
		"upload from response without specification": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(uploadWorkflowSpecServerPath, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
				},
			},
			args:      []string{"-w", "my_workflow"},
			wantError: true,
			expected: []string{
				"workflow specification response is missing specification",
			},
		},
		"unexisting file": {
			args:      []string{"-w", "my_workflow", "non_existing"},
			wantError: true,
			expected: []string{
				"no such file or directory",
			},
		},
		"unexisting workflow": {
			args:      []string{},
			wantError: true,
			expected: []string{
				"workflow name must be provided",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "upload"
			testCmdRun(t, params)
		})
	}
}

func TestUploadContinuesAfterIndividualFailures(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"fail-first.txt",
		"success-one.txt",
		"fail-middle.txt",
		"success-two.txt",
		"fail-last.txt",
	}
	files := make([]string, 0, len(names))
	for _, name := range names {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}

	var mu sync.Mutex
	requestedFiles := make([]string, 0, len(files))
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost ||
				r.URL.Path != "/api/workflows/my_workflow/workspace" {
				t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer 1234" {
				t.Errorf("Expected Authorization Bearer 1234, got %q", got)
			}
			if r.URL.Query().Has("access_token") {
				t.Errorf("Access token leaked into query: %s", r.URL.RawQuery)
			}

			name := filepath.Base(r.URL.Query().Get("file_name"))
			mu.Lock()
			requestedFiles = append(requestedFiles, name)
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if strings.HasPrefix(name, "fail-") {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(
					w,
					`{"message":"upload failed for %s"}`,
					name,
				)
				return
			}
			_, _ = w.Write([]byte(`{"message":"uploaded"}`))
		},
	))
	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		viper.Reset()
		server.Close()
	})

	rootCmd := NewRootCmd()
	args := []string{"upload", "-t", "1234", "-w", "my_workflow"}
	output, err := ExecuteCommand(rootCmd, append(args, files...)...)
	if !errors.Is(err, config.ErrEmpty) {
		t.Fatalf("Expected aggregated upload error, got %v", err)
	}
	for _, name := range names {
		if !strings.Contains(output, name) {
			t.Errorf("Expected output for %s, got %q", name, output)
		}
	}
	for _, name := range []string{"success-one.txt", "success-two.txt"} {
		if !strings.Contains(output, name+" was successfully uploaded") {
			t.Errorf("Expected successful output for %s, got %q", name, output)
		}
	}
	for _, name := range []string{
		"fail-first.txt",
		"fail-middle.txt",
		"fail-last.txt",
	} {
		if !strings.Contains(output, "upload failed for "+name) {
			t.Errorf("Expected failure output for %s, got %q", name, output)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(requestedFiles, names) {
		t.Errorf("Upload requests = %v, want %v", requestedFiles, names)
	}
}
