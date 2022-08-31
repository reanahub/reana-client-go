/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package start

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"reanahub/reana-client-go/pkg/config"
	"testing"
)

var startPathTemplate = "/api/workflows/%s/start"
var paramsPathTemplate = "/api/workflows/%s/parameters"
var statusPathTemplate = "/api/workflows/%s/status"
var lsPathTemplate = "/api/workflows/%s/workspace"

func TestStart(t *testing.T) {
	// Deactivate the sleep used with the --follow flag
	oldInterval := config.CheckInterval
	config.CheckInterval = 0
	t.Cleanup(func() {
		config.CheckInterval = oldInterval
	})

	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"my_workflow is running",
			},
		},
		"valid options": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/params.json",
				},
			},
			Args: []string{"-w", workflowName, "-o", "CACHE=cache,FROM=from"},
			Expected: []string{
				"my_workflow is running",
			},
		},
		"valid parameters": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/params.json",
				},
			},
			Args: []string{"-w", workflowName, "-p", "data=results/data2.root,events=100"},
			Expected: []string{
				"my_workflow is running",
			},
		},
		"unsupported option": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/params.json",
				},
			},
			Args: []string{"-w", workflowName, "-o", "CACHE=cache,INVALID=invalid"},
			Expected: []string{
				"operational option 'INVALID' not supported",
			},
			WantError: true,
		},
		"option not supported for workflow type": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/params.json",
				},
			},
			Args: []string{"-w", workflowName, "-o", "CACHE=cache,report=report"},
			Expected: []string{
				"operational option 'report' not supported for serial workflows",
			},
			WantError: true,
		},
		"parameter not specified": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/params.json",
				},
			},
			Args: []string{"-w", workflowName, "-p", "data=results/data2.root,invalid=100"},
			Expected: []string{
				"given parameter - invalid, is not in reana.yaml",
				"my_workflow is running",
			},
		},
		"validated options and parameters": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/params_only_data.json",
				},
			},
			Args: []string{
				"-w", workflowName, "-o", "TARGET=translated",
				"-p", "data=results/data2.root,invalid=100,removed=test",
			},
			Expected: []string{
				"given parameter - invalid, is not in reana.yaml",
				"given parameter - removed, is not in reana.yaml",
				"my_workflow is running",
			},
		},
		"workflow already finished": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusForbidden,
					ResponseFile: "testdata/already_finished.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"Workflow my_workflow is already finished and cannot be started again.",
			},
			WantError: true,
		},
		"follow stopped": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(statusPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/status_stopped.json",
				},
			},
			Args: []string{"-w", workflowName, "--follow"},
			Expected: []string{
				"the workflow did not finish",
			},
			WantError: true,
		},
		"follow finished": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
				fmt.Sprintf(statusPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/status_finished.json",
				},
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/ls.json",
				},
			},
			Args: []string{"-w", workflowName, "--follow"},
			Expected: []string{
				"my_workflow is running",
				"my_workflow has been finished",
				"Listing workflow output files...",
				"/api/workflows/my_workflow/workspace/code/gendata.C",
				"/api/workflows/my_workflow/workspace/results/data.root",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.Cmd = NewCmd()
			TestCmdRun(t, params)
		})
	}
}
