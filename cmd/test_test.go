/*
This file is part of REANA.
Copyright (C) 2026 CERN.

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
	"testing"
	"time"

	"github.com/reanahub/reana-client-go/pkg/auth"
	"github.com/reanahub/reana-client-go/pkg/config"

	"github.com/spf13/viper"
)

const passingFeature = `Feature: command
Scenario: workflow passed
  When the workflow is finished
  Then the workflow status should be finished
`

func writeCommandFeature(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func serveTestStatus(w http.ResponseWriter, status string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(
		w,
		`{"name":"analysis.1","status":%q,"progress":{}}`,
		status,
	)
}

func configureTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		server.Close()
		viper.Reset()
	})
}

func requireTestToken(t *testing.T, request *http.Request) {
	t.Helper()
	if got := request.Header.Get("Authorization"); got != "Bearer 1234" {
		t.Errorf("got Authorization %q, want Bearer 1234", got)
	}
	if request.URL.Query().Has("access_token") {
		t.Errorf("access token leaked into query: %s", request.URL.RawQuery)
	}
}

func TestTestCommandUsesStoredOIDCToken(t *testing.T) {
	t.Setenv("REANA_ACCESS_TOKEN", "")
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	feature := writeCommandFeature(t, "stored.feature", passingFeature)
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer stored.jwt.token" {
			t.Errorf("got Authorization %q, want stored token", got)
		}
		if r.URL.Query().Has("access_token") {
			t.Errorf("access token leaked into query: %s", r.URL.RawQuery)
		}
		serveTestStatus(w, "finished")
	})
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(viper.GetString("server-url"), auth.Credentials{
		AccessToken:          "stored.jwt.token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}, true)
	if err != nil {
		t.Fatal(err)
	}

	output, err := ExecuteCommand(
		NewRootCmd(), "test", "-w", "analysis.1", "-n", feature,
	)
	if err != nil {
		t.Fatalf("test command failed: %v", err)
	}
	if !strings.Contains(output, "1 passed, 0 failed") {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestTestCommandUsesRepeatedFeatureFileOptions(t *testing.T) {
	first := writeCommandFeature(t, "first.feature", passingFeature)
	second := writeCommandFeature(t, "second.feature", passingFeature)
	statusCalls := 0
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireTestToken(t, r)
		switch r.URL.Path {
		case "/api/workflows/analysis.1/status":
			statusCalls++
			serveTestStatus(w, "finished")
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	})

	output, err := ExecuteCommand(
		NewRootCmd(),
		"test",
		"-t", "1234",
		"-w", "analysis.1",
		"-n", first,
		"-n", second,
	)
	if err != nil {
		t.Fatalf("test command failed: %v", err)
	}
	for _, expected := range []string{
		fmt.Sprintf("Testing file %q", first),
		fmt.Sprintf("Testing file %q", second),
		"2 passed, 0 failed",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in output, got %q", expected, output)
		}
	}
	// Initial status plus two status assertions in each feature.
	if statusCalls != 5 {
		t.Errorf("got %d status calls, want 5", statusCalls)
	}
}

func TestTestCommandLoadsFeatureFilesFromSpecification(t *testing.T) {
	feature := writeCommandFeature(t, "from-spec.feature", passingFeature)
	specificationCalls := 0
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireTestToken(t, r)
		switch r.URL.Path {
		case "/api/workflows/analysis.1/status":
			serveTestStatus(w, "finished")
		case "/api/workflows/analysis.1/specification":
			specificationCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(
				w,
				`{"parameters":{},"specification":{"tests":{"files":[%q]}}}`,
				feature,
			)
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	})

	output, err := ExecuteCommand(
		NewRootCmd(),
		"test",
		"-t", "1234",
		"-w", "analysis.1",
	)
	if err != nil {
		t.Fatalf("test command failed: %v", err)
	}
	if specificationCalls != 1 {
		t.Errorf("got %d specification calls, want 1", specificationCalls)
	}
	if !strings.Contains(output, fmt.Sprintf("Testing file %q", feature)) {
		t.Errorf("feature from specification missing in output: %q", output)
	}
}

func TestTestCommandRequiresFinishedWorkflow(t *testing.T) {
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireTestToken(t, r)
		serveTestStatus(w, "running")
	})

	_, err := ExecuteCommand(
		NewRootCmd(),
		"test",
		"-t", "1234",
		"-w", "analysis.1",
		"-n", "unused.feature",
	)
	if err == nil || !strings.Contains(err.Error(), "must be finished") {
		t.Fatalf("expected unfinished-workflow error, got %v", err)
	}
}

func TestTestCommandFailsWhenScenarioFails(t *testing.T) {
	feature := writeCommandFeature(t, "failure.feature", `Feature: command
Scenario: workflow failed
  When the workflow is finished
  Then the engine logs should contain "missing text"
`)
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireTestToken(t, r)
		switch r.URL.Path {
		case "/api/workflows/analysis.1/status":
			serveTestStatus(w, "finished")
		case "/api/workflows/analysis.1/logs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"logs":"{\"workflow_logs\":\"present\",\"job_logs\":{},\"engine_specific\":null}"}`,
			))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	})

	output, err := ExecuteCommand(
		NewRootCmd(),
		"test",
		"-t", "1234",
		"-w", "analysis.1",
		"-n", feature,
	)
	if !errors.Is(err, config.ErrEmpty) {
		t.Fatalf("expected silent failure exit, got %v", err)
	}
	for _, expected := range []string{"ERROR", "0 passed, 1 failed"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in output, got %q", expected, output)
		}
	}
}

func TestTestCommandReportsMissingFeatureFile(t *testing.T) {
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireTestToken(t, r)
		serveTestStatus(w, "finished")
	})

	_, err := ExecuteCommand(
		NewRootCmd(),
		"test",
		"-t", "1234",
		"-w", "analysis.1",
		"-n", "missing.feature",
	)
	if err == nil ||
		!strings.Contains(err.Error(), "test file missing.feature not found") {
		t.Fatalf("expected feature-file-not-found error, got %v", err)
	}
}

func TestTestCommandRequiresFeatureFiles(t *testing.T) {
	configureTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireTestToken(t, r)
		switch r.URL.Path {
		case "/api/workflows/analysis.1/status":
			serveTestStatus(w, "finished")
		case "/api/workflows/analysis.1/specification":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"parameters":{},"specification":{}}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	})

	_, err := ExecuteCommand(
		NewRootCmd(),
		"test",
		"-t", "1234",
		"-w", "analysis.1",
	)
	if err == nil || !strings.Contains(err.Error(), "no test files specified") {
		t.Fatalf("expected missing-test-files error, got %v", err)
	}
}
