/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const startSuccessBody = `{"message":"started","status":"running","user":"u",` +
	`"workflow_id":"id","workflow_name":"my_workflow"}`

const restartSuccessBody = `{"message":"started","run_number":"1.1",` +
	`"status":"running","user":"u","workflow_id":"id",` +
	`"workflow_name":"my_workflow"}`

func TestRestartSpecificationReadErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := readRestartSpecification(missing); err == nil ||
		!strings.Contains(err.Error(), "could not be read") {
		t.Fatalf("unexpected read error: %v", err)
	}

	malformed := filepath.Join(t.TempDir(), "malformed.yaml")
	if err := os.WriteFile(
		malformed,
		[]byte("workflow: [unterminated\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := readRestartSpecification(malformed); err == nil ||
		!strings.Contains(err.Error(), "not a valid YAML specification") {
		t.Fatalf("unexpected YAML error: %v", err)
	}
}

// A plain restart reuses the workspace and needs no reana.yaml: the -f flag is
// opt-in, so restart must not fail validating a (defaulted) specification file
// that does not exist in the current directory.
func TestRestartWithoutSpecFileSucceeds(t *testing.T) {
	t.Chdir(t.TempDir()) // an empty directory: no reana.yaml here

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(startSuccessBody))
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(), "restart", "-t", "1234", "-w", "my_workflow",
	)
	if err != nil {
		t.Fatalf("plain restart failed: %v", err)
	}
	if !strings.Contains(out, "my_workflow is running") {
		t.Errorf("expected a running status message, got %q", out)
	}
}

func TestRestartDecodesWorkspaceMutationErrors(t *testing.T) {
	for name, statusCode := range map[string]int{
		"conflict":            http.StatusConflict,
		"service unavailable": http.StatusServiceUnavailable,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewTLSServer(
				withPingFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(statusCode)
					_, _ = w.Write(
						[]byte(`{"message":"workspace mutation failed"}`),
					)
				}),
			)
			defer server.Close()
			savedTestServer(t, server.URL, false)
			viper.Set("server-url", server.URL)
			t.Cleanup(viper.Reset)

			_, err := ExecuteCommand(
				NewRootCmd(), "restart", "-t", "1234", "-w", "my_workflow",
			)
			if err == nil ||
				!strings.Contains(err.Error(), "workspace mutation failed") {
				t.Fatalf("unexpected generated-client error: %v", err)
			}
		})
	}
}

// A restart with -f sends the replacement and parameters in one generated
// multipart request.
func TestRestartPostsReplacementSpecAtomically(t *testing.T) {
	specification := "workflow:\n  type: serial\n  specification:\n    steps: []\n"
	specFile := filepath.Join(t.TempDir(), "myreana.yaml")
	if err := os.WriteFile(
		specFile,
		[]byte(specification),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	var restartHit bool
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if !strings.HasSuffix(r.URL.Path, "/restart") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			restartHit = true
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				t.Fatal(err)
			}
			file, _, err := r.FormFile("replacement")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = file.Close() }()
			contents, _ := io.ReadAll(file)
			if string(contents) != specification {
				t.Errorf("replacement bytes changed: %q", contents)
			}
			var parameters map[string]any
			if err := json.Unmarshal(
				[]byte(r.FormValue("parameters")), &parameters,
			); err != nil {
				t.Fatal(err)
			}
			if len(parameters["input_parameters"].(map[string]any)) != 0 {
				t.Errorf("unexpected parameters: %#v", parameters)
			}
			_, _ = w.Write([]byte(restartSuccessBody))
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(),
		"restart",
		"-t",
		"1234",
		"-w",
		"my_workflow",
		"-f",
		specFile,
	)
	if err != nil {
		t.Fatalf("restart -f failed: %v", err)
	}
	if !restartHit {
		t.Error("expected one atomic restart request")
	}
}

func TestRestartValidatesOverridesAgainstReplacementSpec(t *testing.T) {
	specFile := filepath.Join(t.TempDir(), "replacement.yaml")
	specContents := "inputs:\n  parameters:\n    new_parameter: default\n" +
		"workflow:\n  type: serial\n  specification:\n    steps: []\n"
	if err := os.WriteFile(specFile, []byte(specContents), 0o644); err != nil {
		t.Fatal(err)
	}

	var (
		parametersEndpointHit bool
		restartParameters     map[string]any
	)
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.HasSuffix(r.URL.Path, "/parameters"):
				parametersEndpointHit = true
				w.WriteHeader(http.StatusInternalServerError)
			case strings.HasSuffix(r.URL.Path, "/restart"):
				_ = r.ParseMultipartForm(1024 * 1024)
				_ = json.Unmarshal(
					[]byte(r.FormValue("parameters")), &restartParameters,
				)
				_, _ = w.Write([]byte(
					`{"message":"started","status":"running","user":"u",` +
						`"workflow_id":"id","workflow_name":"my_workflow",` +
						`"validation_warnings":[{"message":"using latest tag"}]}`,
				))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"restart",
		"-t",
		"1234",
		"-w",
		"my_workflow",
		"-f",
		specFile,
		"-p",
		"new_parameter=overridden",
	)
	if err != nil {
		t.Fatalf("restart with replacement override failed: %v", err)
	}
	if parametersEndpointHit {
		t.Error("restart fetched parameters from the original specification")
	}
	inputParameters, ok := restartParameters["input_parameters"].(map[string]any)
	if !ok {
		t.Fatalf("missing input_parameters: %#v", restartParameters)
	}
	if got := inputParameters["new_parameter"]; got != "overridden" {
		t.Errorf("expected replacement parameter override, got %#v", got)
	}
	if !strings.Contains(out, "using latest tag") {
		t.Errorf("expected validation warning in output, got %q", out)
	}
}

func TestRestartReturnsAtomicOperationFailure(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			specFile := filepath.Join(t.TempDir(), "replacement.yaml")
			if err := os.WriteFile(
				specFile,
				[]byte("workflow:\n  type: serial\n  specification:\n    steps: []\n"),
				0o600,
			); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewTLSServer(
				withPingFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if !strings.HasSuffix(r.URL.Path, "/restart") {
						http.NotFound(w, r)
						return
					}
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"message":"restart failed"}`))
				}),
			)
			defer server.Close()
			savedTestServer(t, server.URL, false)
			viper.Set("server-url", server.URL)
			t.Cleanup(viper.Reset)

			_, err := ExecuteCommand(
				NewRootCmd(),
				"restart",
				"-t",
				"1234",
				"-w",
				"my_workflow",
				"-f",
				specFile,
			)
			if err == nil || !strings.Contains(err.Error(), "restart failed") {
				t.Fatalf("expected restart failure, got %v", err)
			}
		})
	}
}

func TestRestartReturnsNonRunningStatusAsError(t *testing.T) {
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"message":"not started","status":"failed","user":"u",` +
					`"workflow_id":"id","workflow_name":"my_workflow"}`,
			))
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(), "restart", "-t", "1234", "-w", "my_workflow",
	)
	if err == nil || !strings.Contains(err.Error(), "my_workflow has failed") {
		t.Fatalf("expected failed status error, got %v", err)
	}
}
