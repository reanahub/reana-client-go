/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reanahub/reana-client-go/pkg/config"

	"github.com/spf13/viper"
)

var openPathTemplate = "/api/workflows/%s/open/%s"
var infoURL = "/api/info"

func TestOpen(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"success default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, config.InteractiveSessionTypes[0]): {
					statusCode:   http.StatusOK,
					responseFile: "open_jupyter.json",
				},
				infoURL: {
					statusCode:   http.StatusOK,
					responseFile: "info_big.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=session-secret",
				"It could take several minutes to start the interactive session.",
				"Please note that it will be automatically closed after 7 days of inactivity.",
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				if r.URL.Path != fmt.Sprintf(
					openPathTemplate,
					workflowName,
					config.InteractiveSessionTypes[0],
				) {
					return
				}

				var body map[string]any
				reqBody, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Error while reading request body: %v", err)
				}
				if err := json.Unmarshal(reqBody, &body); err != nil {
					t.Fatalf("Error while unmarshalling request body: %v", err)
				}

				if _, ok := body["secret_names"]; ok {
					t.Fatalf(
						"Expected request body not to contain secret_names, got %v",
						body,
					)
				}
			},
		},
		"success no autoclosure": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, config.InteractiveSessionTypes[0]): {
					statusCode:   http.StatusOK,
					responseFile: "open_jupyter.json",
				},
				infoURL: {
					statusCode:   http.StatusOK,
					responseFile: "info_small.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=session-secret",
				"It could take several minutes to start the interactive session.",
			},
		},
		"success empty max_inactivity_time": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, config.InteractiveSessionTypes[0]): {
					statusCode:   http.StatusOK,
					responseFile: "open_jupyter.json",
				},
				infoURL: {
					statusCode:   http.StatusOK,
					responseFile: "info_empty_inactivity_period.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=session-secret",
				"It could take several minutes to start the interactive session.",
			},
		},
		"success extra args": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, "jupyter"): {
					statusCode:   http.StatusOK,
					responseFile: "open_jupyter.json",
				},
				infoURL: {
					statusCode:   http.StatusOK,
					responseFile: "info_big.json",
				},
			},
			args: []string{"-w", workflowName, "-i", "image", "jupyter"},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=session-secret",
				"It could take several minutes to start the interactive session.",
				"Please note that it will be automatically closed after 7 days of inactivity.",
			},
		},
		"success secret names": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, "jupyter"): {
					statusCode:   http.StatusOK,
					responseFile: "open_jupyter.json",
				},
				infoURL: {
					statusCode:   http.StatusOK,
					responseFile: "info_small.json",
				},
			},
			args: []string{
				"-w", workflowName,
				"--secret-name", "alpha",
				"--secret-name", "beta",
			},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=1234",
				"It could take several minutes to start the interactive session.",
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				if r.URL.Path != fmt.Sprintf(
					openPathTemplate,
					workflowName,
					"jupyter",
				) {
					return
				}

				var body map[string]any
				reqBody, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Error while reading request body: %v", err)
				}
				if err := json.Unmarshal(reqBody, &body); err != nil {
					t.Fatalf("Error while unmarshalling request body: %v", err)
				}

				if _, ok := body["image"]; ok {
					t.Fatalf(
						"Expected request body not to contain image, got %v",
						body,
					)
				}

				secretNames, ok := body["secret_names"].([]any)
				if !ok {
					t.Fatalf(
						"Expected secret_names array in request body, got %v",
						body,
					)
				}

				expectedNames := []string{"alpha", "beta"}
				if len(secretNames) != len(expectedNames) {
					t.Fatalf(
						"Expected %d secret names, got %d",
						len(expectedNames),
						len(secretNames),
					)
				}
				for i, name := range expectedNames {
					if secretNames[i] != name {
						t.Fatalf(
							"Expected secret_names[%d] to be %q, got %v",
							i,
							name,
							secretNames[i],
						)
					}
				}
			},
		},
		"success no secrets": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, "jupyter"): {
					statusCode:   http.StatusOK,
					responseFile: "open_jupyter.json",
				},
				infoURL: {
					statusCode:   http.StatusOK,
					responseFile: "info_small.json",
				},
			},
			args: []string{"-w", workflowName, "--no-secrets"},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=1234",
				"It could take several minutes to start the interactive session.",
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				if r.URL.Path != fmt.Sprintf(
					openPathTemplate,
					workflowName,
					"jupyter",
				) {
					return
				}

				var body map[string]any
				reqBody, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Error while reading request body: %v", err)
				}
				if err := json.Unmarshal(reqBody, &body); err != nil {
					t.Fatalf("Error while unmarshalling request body: %v", err)
				}

				secretNames, ok := body["secret_names"].([]any)
				if !ok {
					t.Fatalf(
						"Expected secret_names array in request body, got %v",
						body,
					)
				}
				if len(secretNames) != 0 {
					t.Fatalf(
						"Expected empty secret_names array, got %v",
						secretNames,
					)
				}
			},
		},
		"invalid session type": {
			args: []string{"-w", workflowName, "invalid"},
			expected: []string{
				fmt.Sprintf(
					"invalid value for 'interactive-session-type': 'invalid' is not part of '%s'",
					strings.Join(config.InteractiveSessionTypes, "', '"),
				),
			},
			wantError: true,
		},
		"conflicting secret options": {
			args: []string{
				"-w", workflowName,
				"--secret-name", "alpha",
				"--no-secrets",
			},
			expected: []string{
				"options --no-secrets and --secret-name cannot be used together",
			},
			wantError: true,
		},
		"workflow already open": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, "jupyter"): {
					statusCode:   http.StatusNotFound,
					responseFile: "open_already_open.json",
				},
			},
			args:      []string{"-w", workflowName},
			expected:  []string{"Interactive session is already open"},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "open"
			testCmdRun(t, params)
		})
	}
}

func TestOpenReturnsErrorWhenSessionSecretFetchFails(t *testing.T) {
	workflowName := "my_workflow"
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == fmt.Sprintf(openPathTemplate, workflowName, config.InteractiveSessionTypes[0]):
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(`{"path": "/session1", "info": {}}`),
				)
			case strings.HasSuffix(r.URL.Path, "/interactive-session-secret"):
				w.WriteHeader(http.StatusInternalServerError)
			default:
				t.Errorf("unexpected request to %q", r.URL.Path)
			}
		},
	))
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(),
		"open",
		"-t",
		"1234",
		"-w",
		workflowName,
	)
	if err == nil {
		t.Fatal("expected an error when the session secret fetch fails")
	}
}
