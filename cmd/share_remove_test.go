/*
This file is part of REANA.
Copyright (C) 2023 CERN.

REANA is free software; you can redistribute it and/or modify it under the terms
of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"testing"
)

var shareRemovePathTemplate = "/api/workflows/%s/unshare"

func TestShareRemove(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareRemovePathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{"-w", workflowName, "--user", "bob@cern.ch"},
			expected: []string{
				"my_workflow is no longer shared with bob@cern.ch",
			},
		},
		"invalid workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareRemovePathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{
				"-w", "invalid",
				"--user", "bob@cern.ch",
			},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "share-remove"
			testCmdRun(t, params)
		})
	}
}
