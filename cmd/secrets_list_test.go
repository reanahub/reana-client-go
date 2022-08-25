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
	successResponse := `[
		{
			"name": "secret1",
			"type": "env"
		},
		{
			"name": "secret2",
			"type": "file"
		}
	]`

	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				secretsListServerPath: {
					statusCode: http.StatusOK,
					body:       successResponse,
				},
			},
			expected: []string{
				"NAME", "TYPE",
				"secret1", "env",
				"secret2", "file",
			},
		},
		"unexpected args": {
			args:      []string{"arg"},
			expected:  []string{"unknown command \"arg\" for \"reana-client secrets-list\""},
			wantError: true,
		},
		"server error": {
			serverResponses: map[string]ServerResponse{
				secretsListServerPath: {
					statusCode: http.StatusInternalServerError,
					body:       `{"message": "Error while querying"}`,
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
