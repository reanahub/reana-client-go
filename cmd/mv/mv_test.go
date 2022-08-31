/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package mv

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var movePathTemplate = "/api/workflows/move_files/%s"

func TestMv(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"success": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(movePathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", "my_workflow", "good/", "new/"},
			Expected: []string{
				"good/ was successfully moved to new/",
			},
		},
		"server error": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(movePathTemplate, workflowName): {
					StatusCode:   http.StatusConflict,
					ResponseFile: "testdata/invalid_path.json",
				},
			},
			Args: []string{"-w", "my_workflow", "bad/", "new/"},
			Expected: []string{
				"Path bad/ does not exists",
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
