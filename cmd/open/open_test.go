/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package open

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"
)

var openPathTemplate = "/api/workflows/%s/open/%s"

func TestOpen(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"success default": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, config.InteractiveSessionTypes[0]): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=1234",
				"It could take several minutes to start the interactive session.",
			},
		},
		"success extra args": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, "jupyter"): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "-i", "image", "jupyter"},
			Expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=1234",
				"It could take several minutes to start the interactive session.",
			},
		},
		"invalid session type": {
			Args: []string{"-w", workflowName, "invalid"},
			Expected: []string{
				fmt.Sprintf(
					"invalid value for 'interactive-session-type': 'invalid' is not part of '%s'",
					strings.Join(config.InteractiveSessionTypes, "', '"),
				),
			},
			WantError: true,
		},
		"workflow already open": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(openPathTemplate, workflowName, "jupyter"): {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "testdata/already_open.json",
				},
			},
			Args:      []string{"-w", workflowName},
			Expected:  []string{"Interactive session is already open"},
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
