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
	successResponse := `{
		"message": "test",
		"workflow_id": "my_workflow_id",
		"workflow_name": "my_workflow"
	}`
	errorResponse := `{
		"message": "Path bad/ does not exists"
	}`

	tests := map[string]TestCmdParams{
		"success": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(movePathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       successResponse,
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
					statusCode: http.StatusConflict,
					body:       errorResponse,
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
