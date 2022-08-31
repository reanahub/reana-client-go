/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package secrets_delete

import (
	"errors"
	"net/http"
	"reanahub/reana-client-go/client/operations"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var secretsDeleteServerPath = "/api/secrets/"

func TestSecretsDelete(t *testing.T) {
	tests := map[string]TestCmdParams{
		"valid secrets": {
			ServerResponses: map[string]ServerResponse{
				secretsDeleteServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"secret1", "secret2"},
			Expected: []string{
				"Secrets secret1, secret2 were successfully deleted",
			},
		},
		"no args": {
			Args: []string{},
			Expected: []string{
				"requires at least 1 arg(s), only received 0",
			},
			WantError: true,
		},
		"invalid secrets": {
			ServerResponses: map[string]ServerResponse{
				secretsDeleteServerPath: {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"secret1", "secret2"},
			Expected: []string{
				"secrets ['secret1', 'secret2'] do not exist. Nothing was deleted",
			},
			WantError: true,
		},
		"valid and invalid secrets": {
			ServerResponses: map[string]ServerResponse{
				secretsDeleteServerPath: {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "testdata/not_found.json",
				},
			},
			Args: []string{"secret1", "secret2"},
			Expected: []string{
				"secrets ['secret1'] do not exist. Nothing was deleted",
			},
			WantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.Cmd = NewCmd()
			TestCmdRun(t, params)
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
			expected: "secrets ['secret1', 'secret2'] do not exist. Nothing was deleted",
		},
		"another error": {
			err:      errors.New("some error"),
			expected: "some error",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := handleDeleteSecretsError(test.err)
			if got.Error() != test.expected {
				t.Errorf("got %s, wanted %s", got.Error(), test.expected)
			}
		})
	}
}
