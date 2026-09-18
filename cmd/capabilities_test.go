/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// legacyServer stands up a released server: its ping omits api_capabilities and
// every bundle endpoint is absent, exactly as before the protocol existed.
func legacyServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	paths := []string{}
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Not found"}`))
		}, legacyPingBody),
	)
	t.Cleanup(server.Close)
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)
	return server, &paths
}

func TestLegacyServerIsRefusedBeforeAnyUpload(t *testing.T) {
	reanaFile := writeSerialSpec(t)
	replacement := filepath.Join(t.TempDir(), "replacement.yaml")
	if err := os.WriteFile(
		replacement,
		[]byte("workflow:\n  type: serial\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	for name, args := range map[string][]string{
		"create":   {"create", "-t", "1234", "-f", reanaFile},
		"validate": {"validate", "-t", "1234", "-f", reanaFile},
		"run":      {"run", "-t", "1234", "-f", reanaFile},
		"restart-with-replacement": {
			"restart", "-t", "1234", "-w", "my_workflow", "-f", replacement,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, paths := legacyServer(t)

			out, err := ExecuteCommand(NewRootCmd(), args...)
			if err == nil {
				t.Fatalf("expected a refusal, got %q", out)
			}
			if !strings.Contains(
				err.Error(),
				workflowSpecificationBundlesCapability,
			) {
				t.Errorf("expected the capability in the error, got %v", err)
			}
			if !strings.Contains(err.Error(), "upgrade the REANA cluster") {
				t.Errorf("expected upgrade guidance, got %v", err)
			}
			// The refusal happens after /api/ping and before any operation, so
			// the bundle is never built or uploaded.
			if len(*paths) != 0 {
				t.Errorf("expected no operation request, got %v", *paths)
			}
		})
	}
}

func TestRestartWithoutReplacementDoesNotRequireTheCapability(t *testing.T) {
	t.Chdir(t.TempDir()) // an empty directory: no reana.yaml here

	var mu sync.Mutex
	pinged := false
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/ping" {
				mu.Lock()
				pinged = true
				mu.Unlock()
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(startSuccessBody))
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	// A restart without --file stays on the compatible /start operation, so it
	// must keep working against a server that never advertises the capability.
	if _, err := ExecuteCommand(
		NewRootCmd(), "restart", "-t", "1234", "-w", "my_workflow",
	); err != nil {
		t.Fatalf("plain restart failed: %v", err)
	}
	if pinged {
		t.Error("plain restart must not require the bundle capability")
	}
}

func TestPingFailureIsReportedWithoutUploading(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	var mu sync.Mutex
	paths := []string{}
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			w.WriteHeader(http.StatusBadGateway)
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
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
		t.Fatal("expected the ping failure to be reported")
	}
	if !strings.Contains(
		err.Error(),
		"could not check REANA server capabilities",
	) {
		t.Errorf("expected a capability-check error, got %v", err)
	}
	for _, path := range paths {
		if path != "/api/ping" {
			t.Errorf(
				"nothing must be uploaded after a failed ping, got %q",
				path,
			)
		}
	}
}
