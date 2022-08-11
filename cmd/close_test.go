/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"testing"
)

var closePathTemplate = "/api/workflows/%s/close/"

func TestClose(t *testing.T) {
	tests := map[string]struct {
		serverResponse string
		status         int
		expected       []string
		args           []string
		workflow       string
	}{
		"success": {
			serverResponse: "{}",
			status:         http.StatusOK,
			expected: []string{
				"Interactive session for workflow my_workflow was successfully closed",
			},
			args:     []string{"-w", "my_workflow"},
			workflow: "my_workflow",
		},
		"error": {
			serverResponse: `{"message": "Workflow - my_workflow has no open interactive session."}`,
			status:         http.StatusNotFound,
			expected:       []string{"Workflow - my_workflow has no open interactive session."},
			args:           []string{"-w", "my_workflow"},
			workflow:       "my_workflow",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			closeServerPath := fmt.Sprintf(closePathTemplate, test.workflow)
			testCmdRun(
				t,
				"close",
				closeServerPath,
				test.serverResponse,
				test.status,
				test.expected,
				test.args...)
		})
	}
}
