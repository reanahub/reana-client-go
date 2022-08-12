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

var duPathTemplate = "/api/workflows/%s/disk_usage"

func TestDu(t *testing.T) {
	workflowName := "my_workflow"
	successResponse := `{
		"disk_usage_info": [
			{
				"name": "/code/fitdata.C",
				"size": {
					"human_readable": "2 KiB",
					"raw": 2048
				}
			},
			{
				"name": "/code/gendata.C",
				"size": {
					"human_readable": "4.5 KiB",
					"raw": 4608
				}
			}
		],
		"user": "user",
		"workflow_id": "my_workflow_id",
		"workflow_name": "my_workflow"
	}`
	tests := map[string]TestCmdParams{
		"default": {
			serverPath:     fmt.Sprintf(duPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName},
			expected: []string{
				"SIZE", "NAME",
				"2048", "./code/fitdata.C",
				"4608", "./code/gendata.C",
			},
		},
		"summarize": {
			serverPath: fmt.Sprintf(duPathTemplate, workflowName),
			serverResponse: `{
				"disk_usage_info": [
					{
						"size": {
							"human_readable": "10 KiB",
							"raw": 10240
						}
					}
				],
				"user": "user",
				"workflow_id": "my_workflow_id",
				"workflow_name": "my_workflow"
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName, "-s"},
			expected: []string{
				"SIZE", "NAME",
				"10240", ".",
			},
		},
		"human readable": {
			serverPath:     fmt.Sprintf(duPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "-r"},
			expected: []string{
				"SIZE", "NAME",
				"2 KiB", "./code/fitdata.C",
				"4.5 KiB", "./code/gendata.C",
			},
		},
		"files in black list": {
			serverPath: fmt.Sprintf(duPathTemplate, workflowName),
			serverResponse: `{
				"disk_usage_info": [
					{
						"name": ".git/test.C",
						"size": {
							"human_readable": "2 KiB",
							"raw": 2048
						}
					},
					{
						"name": "/code/gendata.C",
						"size": {
							"human_readable": "4.5 KiB",
							"raw": 4608
						}
					}
				],
				"user": "user",
				"workflow_id": "my_workflow_id",
				"workflow_name": "my_workflow"
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName},
			expected: []string{
				"SIZE", "NAME",
				"4608", "./code/gendata.C",
			},
			unwanted: []string{
				"2048", "./git/test.C",
			},
		},
		"with filters": {
			serverPath: fmt.Sprintf(duPathTemplate, workflowName),
			serverResponse: `{
				"disk_usage_info": [
					{
						"name": "/code/gendata.C",
						"size": {
							"human_readable": "4.5 KiB",
							"raw": 2048
						}
					}
				],
				"user": "user",
				"workflow_id": "my_workflow_id",
				"workflow_name": "my_workflow"
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName, "--filter", "name=./code/gendata.C,size=2048"},
			expected: []string{
				"SIZE", "NAME",
				"2048", "./code/gendata.C",
			},
		},
		"malformed filters": {
			serverPath: fmt.Sprintf(duPathTemplate, workflowName),
			args:       []string{"-w", workflowName, "--filter", "name"},
			expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			wantError: true,
		},
		"no matching files:": {
			serverPath: fmt.Sprintf(duPathTemplate, workflowName),
			serverResponse: `{
				"disk_usage_info": [],
				"user": "user",
				"workflow_id": "my_workflow_id",
				"workflow_name": "my_workflow"
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName, "--filter", "name=nothing.C"},
			expected:   []string{"no files matching filter criteria"},
			wantError:  true,
		},
		"unexisting workflow": {
			serverPath:     fmt.Sprintf(duPathTemplate, "invalid"),
			serverResponse: `{"message": "REANA_WORKON is set to invalid, but that workflow does not exist."}`,
			statusCode:     http.StatusNotFound,
			args:           []string{"-w", "invalid"},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "du"
			testCmdRun(t, params)
		})
	}
}
