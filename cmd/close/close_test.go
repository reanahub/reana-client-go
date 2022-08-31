/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package close

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var closePathTemplate = "/api/workflows/%s/close/"

func TestClose(t *testing.T) {
	tests := map[string]TestCmdParams{
		"success": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(closePathTemplate, "my_workflow"): {
					StatusCode:   http.StatusOK,
					ResponseFile: "../../testdata/inputs/empty.json",
				},
			},
			Args: []string{"-w", "my_workflow"},
			Expected: []string{
				"Interactive session for workflow my_workflow was successfully closed",
			},
		},
		"error": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(closePathTemplate, "my_workflow"): {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "testdata/no_open.json",
				},
			},
			Args:      []string{"-w", "my_workflow"},
			Expected:  []string{"Workflow - my_workflow has no open interactive session."},
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
