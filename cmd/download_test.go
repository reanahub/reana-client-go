/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var downloadWorkflowSpecServerPath = "/api/workflows/%s/specification"
var downloadServerPath = "/api/workflows/%s/workspace/%s"

func TestFileDownload(t *testing.T) {

	fileName := "results/plot.png"
	dirName := "results"
	dirZipFileName := "download_roofit.1_results_2022-10-03-122917.zip"
	batchDir := t.TempDir()
	writeFailureDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(writeFailureDir, "blocked"), 0o700); err != nil {
		t.Fatal(err)
	}

	tests := map[string]TestCmdParams{
		"download file specified in the workflow specification as outputs": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadWorkflowSpecServerPath, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "workflow_specification.json",
				},
				fmt.Sprintf(downloadServerPath, "my_workflow", fileName): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": fmt.Sprintf(
							`attachment; filename="%s"`,
							fileName,
						),
					},
				},
			},
			args: []string{"-w", "my_workflow"},
			expected: []string{
				fmt.Sprintf("%s was successfully downloaded.", fileName),
			},
		},
		"download from workflow without outputs": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadWorkflowSpecServerPath, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "workflow_specification_without_outputs.json",
				},
			},
			args: []string{"-w", "my_workflow"},
		},
		"download from response without specification": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadWorkflowSpecServerPath, "my_workflow"): {
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
		"download file specified as argument": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadServerPath, "my_workflow", fileName): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": fmt.Sprintf(
							`attachment; filename="%s"`,
							fileName,
						),
					},
				},
			},
			args: []string{"-w", "my_workflow", fileName},
			expected: []string{
				fmt.Sprintf("%s was successfully downloaded.", fileName),
			},
		},
		"download directory specified as argument": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadServerPath, "my_workflow", dirName): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": fmt.Sprintf(
							`attachment; filename="%s"`,
							dirZipFileName,
						),
					},
				},
			},
			args: []string{"-w", "my_workflow", dirName, "-o", dirName},
			expected: []string{
				fmt.Sprintf("%s was successfully downloaded.", dirZipFileName),
			},
		},
		"download unexisting file": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadServerPath, "my_workflow", "file"): {
					statusCode:   http.StatusNotFound,
					responseFile: "download_file_not_found.json",
				},
			},
			args:      []string{"-w", "my_workflow", "file"},
			wantError: true,
			expected: []string{
				"file does not exist.",
			},
		},
		"continue after individual download failures": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"fail-first.txt",
				): {
					statusCode:   http.StatusNotFound,
					responseFile: "download_file_not_found.json",
				},
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"success-one.txt",
				): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": `attachment; filename="success-one.txt"`,
					},
				},
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"fail-middle.txt",
				): {
					statusCode:   http.StatusNotFound,
					responseFile: "download_file_not_found.json",
				},
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"success-two.txt",
				): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": `attachment; filename="success-two.txt"`,
					},
				},
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"fail-last.txt",
				): {
					statusCode:   http.StatusNotFound,
					responseFile: "download_file_not_found.json",
				},
			},
			args: []string{
				"-w", "my_workflow",
				"fail-first.txt",
				"success-one.txt",
				"fail-middle.txt",
				"success-two.txt",
				"fail-last.txt",
				"-o", batchDir,
			},
			expected: []string{
				"File fail-first.txt could not be downloaded: file does not exist",
				"File success-one.txt was successfully downloaded",
				"File fail-middle.txt could not be downloaded: file does not exist",
				"File success-two.txt was successfully downloaded",
				"File fail-last.txt could not be downloaded: file does not exist",
			},
			wantError: true,
		},
		"continue after local write failure": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"write-failure.txt",
				): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": `attachment; filename="blocked"`,
					},
				},
				fmt.Sprintf(
					downloadServerPath,
					"my_workflow",
					"success-after-write-failure.txt",
				): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": `attachment; filename="success-after-write-failure.txt"`,
					},
				},
			},
			args: []string{
				"-w", "my_workflow",
				"write-failure.txt",
				"success-after-write-failure.txt",
				"-o", writeFailureDir,
			},
			expected: []string{
				"File blocked could not be written",
				"File success-after-write-failure.txt was successfully downloaded",
			},
			wantError: true,
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
			params.cmd = "download"
			testCmdRun(t, params)
		})
	}
	t.Cleanup(func() {
		// Remove all the temp files created by the test
		err := os.RemoveAll(dirName)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func TestDownloadToStdoutKeepsDiagnosticsOnStderr(t *testing.T) {
	const fileContent = "downloaded file content"
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET request, got %s", r.Method)
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer 1234" {
				t.Errorf("Expected Authorization Bearer 1234, got %q", got)
			}
			if r.URL.Query().Has("access_token") {
				t.Errorf("Access token leaked into query: %s", r.URL.RawQuery)
			}
			switch r.URL.Path {
			case "/api/workflows/my_workflow/workspace/missing.txt":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"file does not exist."}`))
			case "/api/workflows/my_workflow/workspace/invalid.zip":
				w.Header().Set("Content-Type", "application/zip")
				w.Header().Set(
					"Content-Disposition",
					`attachment; filename="invalid.zip"`,
				)
				_, _ = w.Write([]byte("not a zip archive"))
			case "/api/workflows/my_workflow/workspace/success.txt":
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set(
					"Content-Disposition",
					`attachment; filename="success.txt"`,
				)
				_, _ = w.Write([]byte(fileContent))
			default:
				t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
			}
		},
	))
	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		viper.Reset()
		server.Close()
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd := NewRootCmd()
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"download", "-t", "1234", "-w", "my_workflow",
		"missing.txt", "invalid.zip", "success.txt", "-o", "-",
	})
	err := rootCmd.Execute()
	if !errors.Is(err, config.ErrEmpty) {
		t.Fatalf("Expected aggregated download error, got %v", err)
	}
	if got := stdout.String(); got != fileContent {
		t.Errorf("Standard output = %q, want %q", got, fileContent)
	}
	if !strings.Contains(
		stderr.String(),
		"File missing.txt could not be downloaded: file does not exist.",
	) {
		t.Errorf(
			"Expected download failure on standard error, got %q",
			stderr.String(),
		)
	}
	if !strings.Contains(
		stderr.String(),
		"File invalid.zip could not be downloaded:",
	) {
		t.Errorf(
			"Expected archive failure on standard error, got %q",
			stderr.String(),
		)
	}
	if strings.Contains(stderr.String(), "invalid.zip could not be written") {
		t.Errorf(
			"Archive failure was reported as a write error: %q",
			stderr.String(),
		)
	}
}
