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

var rmPathTemplate = "/api/workflows/%s/workspace/%s"

func TestRm(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"multiple files": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(rmPathTemplate, workflowName, "files/*"): {
					statusCode:   http.StatusOK,
					responseFile: "rm.json",
				},
			},
			args: []string{"-w", workflowName, "files/*"},
			expected: []string{
				"File files/one.py was successfully deleted",
				"File files/two.py was successfully deleted",
				"Something went wrong while deleting files/three.py",
				"testing error in three.py",
				"60 bytes freed up",
			},
			wantError: true,
		},
		"no space freed": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(rmPathTemplate, workflowName, "files/*"): {
					statusCode:   http.StatusOK,
					responseFile: "rm_no_freed.json",
				},
			},
			args: []string{"-w", workflowName, "files/*"},
			expected: []string{
				"File files/empty.py was successfully deleted",
			},
			unwanted: []string{
				"Something went wrong while deleting",
				"bytes freed up",
			},
		},
		"no matching files": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(rmPathTemplate, workflowName, "files/*"): {
					statusCode:   http.StatusOK,
					responseFile: "rm_empty.json",
				},
			},
			args: []string{"-w", workflowName, "files/*"},
			expected: []string{
				"files/* did not match any existing file",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "rm"
			testCmdRun(t, params)
		})
	}
}
