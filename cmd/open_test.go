/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"
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
				"/test/jupyter?token=1234",
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
				"/test/jupyter?token=1234",
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
				"/test/jupyter?token=1234",
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
				"/test/jupyter?token=1234",
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
