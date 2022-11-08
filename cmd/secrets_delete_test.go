/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"net/http"
	"reanahub/reana-client-go/client/operations"
	"testing"
)

var secretsDeleteServerPath = "/api/secrets/"

func TestSecretsDelete(t *testing.T) {
	tests := map[string]TestCmdParams{
		"valid secrets": {
			serverResponses: map[string]ServerResponse{
				secretsDeleteServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "secrets_delete_multiple.json",
				},
			},
			args: []string{"secret1", "secret2"},
			expected: []string{
				"Secrets secret1, secret2 were successfully deleted",
			},
		},
		"no args": {
			args: []string{},
			expected: []string{
				"requires at least 1 arg(s), only received 0",
			},
			wantError: true,
		},
		"invalid secrets": {
			serverResponses: map[string]ServerResponse{
				secretsDeleteServerPath: {
					statusCode:   http.StatusNotFound,
					responseFile: "secrets_delete_multiple.json",
				},
			},
			args: []string{"secret1", "secret2"},
			expected: []string{
				"secrets secret1, secret2 do not exist. Nothing was deleted",
			},
			wantError: true,
		},
		"valid and invalid secrets": {
			serverResponses: map[string]ServerResponse{
				secretsDeleteServerPath: {
					statusCode:   http.StatusNotFound,
					responseFile: "secrets_delete_single.json",
				},
			},
			args: []string{"secret1", "secret2"},
			expected: []string{
				"secrets secret1 do not exist. Nothing was deleted",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "secrets-delete"
			testCmdRun(t, params)
		})
	}
}

func TestHandleSecretsDeleteApiError(t *testing.T) {
	notFoundError := operations.NewDeleteSecretsNotFound()
	notFoundError.Payload = []string{"secret1", "secret2"}

	tests := map[string]struct {
		err      error
		expected string
	}{
		"not found error": {
			err:      notFoundError,
			expected: "secrets secret1, secret2 do not exist. Nothing was deleted",
		},
		"another error": {
			err:      errors.New("some error"),
			expected: "some error",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := handleSecretsDeleteApiError(test.err)
			if got.Error() != test.expected {
				t.Errorf("got %s, wanted %s", got.Error(), test.expected)
			}
		})
	}
}
