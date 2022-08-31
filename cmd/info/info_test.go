/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package info

import (
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var infoServerPath = "/api/info"

func TestInfo(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				infoServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Expected: []string{
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
			ServerResponses: map[string]ServerResponse{
				infoServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"--json"},
			Expected: []string{
				"\"compute_backends\": {", "\"value\": [", "\"kubernetes\",",
				"\"workspaces_available\": {", "\"/var/reana\",", "\"/var/cern\"",
				"\"title\": \"List of available workspaces\",",
			},
		},
		"missing fields": {
			ServerResponses: map[string]ServerResponse{
				infoServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/minimal.json",
				},
			},
			Expected: []string{
				"Maximum allowed memory limit for Kubernetes jobs: None",
				"Maximum retention period in days for workspace files: None",
			},
			Unwanted: []string{
				"List of supported compute backends", "Default workspace",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.Cmd = NewCmd()
			TestCmdRun(t, params)
		})
	}
}
