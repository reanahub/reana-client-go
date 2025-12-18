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

var shareStatusPathTemplate = "/api/workflows/%s/share-status"

func TestShareStatus(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareStatusPathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				fmt.Sprintf(
					"Workflow %s is not shared with anyone.",
					workflowName,
				),
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "share-status"
			testCmdRun(t, params)
		})
	}
}
