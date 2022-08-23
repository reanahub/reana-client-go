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
	successResponse := `{
	"compute_backends": {
		"title": "List of supported compute backends",
		"value": [
			"kubernetes",
			"slurmcern"
		]
	},
	"default_kubernetes_jobs_timeout": {
		"title": "Default timeout for Kubernetes jobs",
		"value": "124"
	},
	"default_kubernetes_memory_limit": {
		"title": "Default memory limit for Kubernetes jobs",
		"value": "248"
	},
	"default_workspace": {
		"title": "Default workspace",
		"value": "/var/reana"
	},
	"kubernetes_max_memory_limit": {
		"title": "Maximum allowed memory limit for Kubernetes jobs",
		"value": "1000"
	},
	"maximum_kubernetes_jobs_timeout": {
		"title": "Maximum timeout for Kubernetes jobs",
		"value": "500"
	},
	"maximum_workspace_retention_period": {
		"title": "Maximum retention period in days for workspace files",
		"value": "250"
	},
	"workspaces_available": {
		"title": "List of available workspaces",
		"value": [
			"/var/reana",
			"/var/cern"
		]
	}
}
`
	minimalResponse := `{
	"kubernetes_max_memory_limit": {
		"title": "Maximum allowed memory limit for Kubernetes jobs"
	},
	"maximum_workspace_retention_period": {
		"title": "Maximum retention period in days for workspace files"
	}
}
`

	tests := map[string]TestCmdParams{
		"default": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
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
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--json"},
			expected: []string{
				"\"compute_backends\": {", "\"value\": [", "\"kubernetes\",",
				"\"workspaces_available\": {", "\"/var/reana\",", "\"/var/cern\"",
				"\"title\": \"List of available workspaces\",",
			},
		},
		"missing fields": {
			serverResponse: minimalResponse,
			statusCode:     http.StatusOK,
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
			params.serverPath = infoServerPath
			testCmdRun(t, params)
		})
	}
}
