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
	"reanahub/reana-client-go/pkg/config"
	"testing"
)

var startPathTemplate = "/api/workflows/%s/start"
var paramsPathTemplate = "/api/workflows/%s/parameters"

func TestStart(t *testing.T) {
	// Deactivate the sleep used with the --follow flag
	oldInterval := config.CheckInterval
	config.CheckInterval = 0
	t.Cleanup(func() {
		config.CheckInterval = oldInterval
	})

	workflowName := "my_workflow"
	startResponse := `{
		"message": "Workflow successfully started",
		"status": "running",
		"user": "user",
		"workflow_id": "my_workflow_id",
		"workflow_name": "my_workflow"
	}`
	paramsResponse := `{
		"id": "my_workflow_id",
		"name": "my_workflow",
		"type": "serial",
		"parameters": {
			"data": "results/data.root",
			"events": 20,
			"plot": "results/plot.png"
		}
	}`
	statusFinishedResponse := `{
		"status": "finished",
		"user": "user",
		"workflow_id": "my_workflow_id",
		"workflow_name": "my_workflow",
		"created": "2022-07-20T12:08:40"
	}`
	statusStoppedResponse := `{
		"status": "stopped",
		"user": "user",
		"workflow_id": "my_workflow_id",
		"workflow_name": "my_workflow",
		"created": "2022-07-20T12:08:40"
	}`
	lsResponse := `{
		"items": [
			{
				"last-modified": "2022-07-11T12:50:33",
				"name": "code/gendata.C",
				"size": {
					"human_readable": "1.89 KiB",
					"raw": 1937
				}
			}
		],
		"total": 1
}`

	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"my_workflow is running",
			},
		},
		"valid options": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       paramsResponse,
				},
			},
			args: []string{"-w", workflowName, "-o", "CACHE=cache,FROM=from"},
			expected: []string{
				"my_workflow is running",
			},
		},
		"valid parameters": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       paramsResponse,
				},
			},
			args: []string{"-w", workflowName, "-p", "data=results/data2.root,events=100"},
			expected: []string{
				"my_workflow is running",
			},
		},
		"unsupported option": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       paramsResponse,
				},
			},
			args: []string{"-w", workflowName, "-o", "CACHE=cache,INVALID=invalid"},
			expected: []string{
				"operational option 'INVALID' not supported",
			},
			wantError: true,
		},
		"option not supported for workflow type": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       paramsResponse,
				},
			},
			args: []string{"-w", workflowName, "-o", "CACHE=cache,report=report"},
			expected: []string{
				"operational option 'report' not supported for serial workflows",
			},
			wantError: true,
		},
		"parameter not specified": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       paramsResponse,
				},
			},
			args: []string{"-w", workflowName, "-p", "data=results/data2.root,invalid=100"},
			expected: []string{
				"given parameter - invalid, is not in reana.yaml",
				"my_workflow is running",
			},
		},
		"validated options and parameters": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(paramsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body: `{
						"id": "my_workflow_id",
						"name": "my_workflow",
						"type": "cwl",
						"parameters": {
							"data": "results/data.root"
						}
					}`,
				},
			},
			args: []string{
				"-w", workflowName, "-o", "TARGET=translated",
				"-p", "data=results/data2.root,invalid=100,removed=test",
			},
			expected: []string{
				"given parameter - invalid, is not in reana.yaml",
				"given parameter - removed, is not in reana.yaml",
				"my_workflow is running",
			},
		},
		"workflow already finished": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusForbidden,
					body:       `{"message": "Workflow my_workflow is already finished and cannot be started again."}`,
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"Workflow my_workflow is already finished and cannot be started again.",
			},
			wantError: true,
		},
		"follow stopped": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(statusPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       statusStoppedResponse,
				},
			},
			args: []string{"-w", workflowName, "--follow"},
			expected: []string{
				"the workflow did not finish",
			},
			wantError: true,
		},
		"follow finished": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(startPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       startResponse,
				},
				fmt.Sprintf(statusPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       statusFinishedResponse,
				},
				fmt.Sprintf(lsPathTemplate, workflowName): {
					statusCode: http.StatusOK,
					body:       lsResponse,
				},
			},
			args: []string{"-w", workflowName, "--follow"},
			expected: []string{
				"my_workflow is running",
				"my_workflow has been finished",
				"Listing workflow output files...",
				"/api/workflows/my_workflow/workspace/code/gendata.C",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "start"
			testCmdRun(t, params)
		})
	}
}
