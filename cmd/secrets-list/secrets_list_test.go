/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package secrets_list

import (
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var secretsListServerPath = "/api/secrets"

func TestSecretsList(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				secretsListServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Expected: []string{
				"NAME", "TYPE",
				"secret1", "env",
				"secret2", "file",
			},
		},
		"unexpected args": {
			Args:      []string{"arg"},
			Expected:  []string{"unknown command \"arg\" for \"secrets-list\""},
			WantError: true,
		},
		"server error": {
			ServerResponses: map[string]ServerResponse{
				secretsListServerPath: {
					StatusCode:   http.StatusInternalServerError,
					ResponseFile: "../../testdata/inputs/internal_server_error.json",
				},
			},
			Expected: []string{
				"Error while querying",
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
