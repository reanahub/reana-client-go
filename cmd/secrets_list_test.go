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

var secretsListServerPath = "/api/secrets"

func TestSecretsList(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				secretsListServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "secrets_list.json",
				},
			},
			expected: []string{
				"NAME", "TYPE",
				"secret1", "env",
				"secret2", "file",
			},
		},
		"unexpected args": {
			args: []string{"arg"},
			expected: []string{
				"unknown command \"arg\" for \"reana-client-go secrets-list\"",
			},
			wantError: true,
		},
		"server error": {
			serverResponses: map[string]ServerResponse{
				secretsListServerPath: {
					statusCode:   http.StatusInternalServerError,
					responseFile: "common_internal_server_error.json",
				},
			},
			expected: []string{
				"Error while querying",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "secrets-list"
			testCmdRun(t, params)
		})
	}
}
