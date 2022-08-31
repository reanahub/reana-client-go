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

var movePathTemplate = "/api/workflows/move_files/%s"

func TestMv(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"success": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(movePathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "mv.json",
				},
			},
			args: []string{"-w", "my_workflow", "good/", "new/"},
			expected: []string{
				"good/ was successfully moved to new/",
			},
		},
		"server error": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(movePathTemplate, workflowName): {
					statusCode:   http.StatusConflict,
					responseFile: "mv_invalid_path.json",
				},
			},
			args: []string{"-w", "my_workflow", "bad/", "new/"},
			expected: []string{
				"Path bad/ does not exists",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "mv"
			testCmdRun(t, params)
		})
	}
}
