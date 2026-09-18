/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRunCreatesUploadsAndStartsWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		localSpec     string
		specification string
		inputFile     string
		wantRequests  []string
	}{
		{
			name:          "without inputs",
			localSpec:     "workflow:\n  type: serial\n",
			specification: `{"parameters":{},"specification":{"workflow":{"type":"serial","specification":{"steps":[]}}}}`,
			wantRequests: []string{
				"POST /api/workflows",
				"GET /api/workflows/analysis.1/specification",
				"POST /api/workflows/analysis.1/start",
			},
		},
		{
			name:          "with declared input",
			localSpec:     "inputs:\n  files:\n    - input.txt\nworkflow:\n  type: serial\n",
			specification: `{"parameters":{},"specification":{"inputs":{"files":["input.txt"],"directories":[]},"workflow":{"type":"serial","specification":{"steps":[]}}}}`,
			inputFile:     "input.txt",
			wantRequests: []string{
				"POST /api/workflows",
				"GET /api/workflows/analysis.1/specification",
				"POST /api/workflows/analysis.1/workspace",
				"POST /api/workflows/analysis.1/start",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if test.inputFile != "" {
				if err := os.WriteFile(
					test.inputFile,
					[]byte("input"),
					0o600,
				); err != nil {
					t.Fatal(err)
				}
			}
			reanaFile := writeSpec(t, test.localSpec)
			var requests []string

			server := httptest.NewTLSServer(
				withPingFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, r.Method+" "+r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodPost && r.URL.Path == "/api/workflows":
						w.WriteHeader(http.StatusCreated)
						_, _ = w.Write(
							[]byte(
								`{"workflow_name":"analysis.1","workflow_id":"id"}`,
							),
						)
					case r.Method == http.MethodGet && r.URL.Path == "/api/workflows/analysis.1/specification":
						_, _ = w.Write([]byte(test.specification))
					case r.Method == http.MethodPost && r.URL.Path == "/api/workflows/analysis.1/workspace":
						if fileName := r.URL.Query().Get("file_name"); fileName != test.inputFile {
							t.Errorf(
								"unexpected uploaded file name: got %q, want %q",
								fileName,
								test.inputFile,
							)
						}
						contents, err := io.ReadAll(r.Body)
						if err != nil {
							t.Errorf(
								"could not read uploaded contents: %v",
								err,
							)
							w.WriteHeader(http.StatusInternalServerError)
							_, _ = w.Write(
								[]byte(`{"message":"test handler failed"}`),
							)
							return
						}
						if string(contents) != "input" {
							t.Errorf(
								"unexpected uploaded contents: got %q, want %q",
								contents,
								"input",
							)
						}
						_, _ = w.Write([]byte(`{"message":"uploaded"}`))
					case r.Method == http.MethodPost && r.URL.Path == "/api/workflows/analysis.1/start":
						_, _ = w.Write(
							[]byte(
								`{"message":"started","status":"running","user":"u","workflow_id":"id","workflow_name":"analysis.1"}`,
							),
						)
					default:
						t.Errorf(
							"unexpected request: %s %s",
							r.Method,
							r.URL.Path,
						)
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write(
							[]byte(`{"message":"test handler failed"}`),
						)
						return
					}
				}),
			)
			defer server.Close()

			savedTestServer(t, server.URL, false)
			viper.Set("server-url", server.URL)
			t.Cleanup(viper.Reset)
			out, err := ExecuteCommand(
				NewRootCmd(),
				"run",
				"-t", "1234",
				"-f", reanaFile,
				"-w", "analysis",
			)
			if err != nil {
				t.Fatalf("run failed: %v", err)
			}

			if !reflect.DeepEqual(requests, test.wantRequests) {
				t.Fatalf(
					"unexpected request sequence: got %v, want %v",
					requests,
					test.wantRequests,
				)
			}
			for _, message := range []string{
				"Creating a workflow...",
				"analysis.1",
				"Uploading files...",
				"Starting workflow...",
				"analysis.1 is running",
			} {
				if !strings.Contains(out, message) {
					t.Errorf("expected %q in output, got %q", message, out)
				}
			}
		})
	}
}

func TestRunStopsWhenCreationFails(t *testing.T) {
	reanaFile := writeSerialSpec(t)
	requestCount := 0
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"invalid workflow"}`))
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)
	_, err := ExecuteCommand(NewRootCmd(), "run", "-t", "1234", "-f", reanaFile)
	if err == nil || !strings.Contains(err.Error(), "invalid workflow") {
		t.Fatalf("expected the creation error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf(
			"run continued after creation failure: got %d requests",
			requestCount,
		)
	}
}
