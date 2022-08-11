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
	serverResponse := `{
  "compute_backends": {
    "value": [
      "kubernetes",
      "slurmcern"
    ]
  },
  "default_workspace": {
    "value": "/var/reana"
  },
  "workspaces_available": {
    "value": [
      "/var/reana",
      "/var/cern"
    ]
  }
}
`

	tests := map[string]struct {
		expected []string
		args     []string
	}{
		"default": {
			expected: []string{
				"List of supported compute backends: kubernetes, slurmcern",
				"Default workspace: /var/reana",
				"List of available workspaces: /var/reana, /var/cern",
			},
		},
		"json": {
			args:     []string{"--json"},
			expected: []string{serverResponse},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testCmdRun(
				t,
				"info",
				infoServerPath,
				serverResponse,
				http.StatusOK,
				test.expected,
				test.args...)
		})
	}
}
