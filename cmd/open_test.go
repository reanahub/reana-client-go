/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"

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
