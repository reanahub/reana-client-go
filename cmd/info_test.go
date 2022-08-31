/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"testing"
)

var infoServerPath = "/api/info"

func TestInfo(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info.json",
				},
			},
			expected: []string{
				"List of supported compute backends: kubernetes, slurmcern",
				"Default timeout for Kubernetes jobs: 124",
				"Default memory limit for Kubernetes jobs: 248",
				"Default workspace: /var/reana",
				"Maximum allowed memory limit for Kubernetes jobs: 1000",
				"Maximum timeout for Kubernetes jobs: 500",
				"Maximum retention period in days for workspace files: 250",
				"List of available workspaces: /var/reana, /var/cern",
			},
		},
		"json": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info.json",
				},
			},
			args: []string{"--json"},
			expected: []string{
				"\"compute_backends\": {", "\"value\": [", "\"kubernetes\",",
				"\"workspaces_available\": {", "\"/var/reana\",", "\"/var/cern\"",
				"\"title\": \"List of available workspaces\",",
			},
		},
		"missing fields": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_minimal.json",
				},
			},
			expected: []string{
				"Maximum allowed memory limit for Kubernetes jobs: None",
				"Maximum retention period in days for workspace files: None",
			},
			unwanted: []string{
				"List of supported compute backends", "Default workspace",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "info"
			testCmdRun(t, params)
		})
	}
}
