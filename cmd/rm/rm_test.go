/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package rm

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var rmPathTemplate = "/api/workflows/%s/workspace/%s"

func TestRm(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"multiple files": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(rmPathTemplate, workflowName, "files/*"): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "files/*"},
			Expected: []string{
				"File files/one.py was successfully deleted",
				"File files/two.py was successfully deleted",
				"Something went wrong while deleting files/three.py",
				"testing error in three.py",
				"60 bytes freed up",
			},
			WantError: true,
		},
		"no space freed": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(rmPathTemplate, workflowName, "files/*"): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_freed.json",
				},
			},
			Args: []string{"-w", workflowName, "files/*"},
			Expected: []string{
				"File files/empty.py was successfully deleted",
			},
			Unwanted: []string{
				"Something went wrong while deleting",
				"bytes freed up",
			},
		},
		"no matching files": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(rmPathTemplate, workflowName, "files/*"): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/empty.json",
				},
			},
			Args: []string{"-w", workflowName, "files/*"},
			Expected: []string{
				"files/* did not match any existing file",
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
