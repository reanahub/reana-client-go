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
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "du_regular_files.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"SIZE", "NAME",
				"2048", "./code/fitdata.C",
				"4608", "./code/gendata.C",
			},
		},
		"summarize": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "du_summarize.json",
				},
			},
			args: []string{"-w", workflowName, "-s"},
			expected: []string{
				"SIZE", "NAME",
				"10240", ".",
			},
		},
		"human readable": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "du_regular_files.json",
				},
			},
			args: []string{"-w", workflowName, "-h"},
			expected: []string{
				"SIZE", "NAME",
				"2 KiB", "./code/fitdata.C",
				"4.5 KiB", "./code/gendata.C",
			},
		},
		"files in black list": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "du_ignored_files.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"SIZE", "NAME",
				"4608", "./code/gendata.C",
			},
			unwanted: []string{
				"2048", "./git/test.C",
			},
		},
		"with filters": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "du_filtered.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--filter",
				"name=./code/gendata.C,size=2048",
			},
			expected: []string{
				"SIZE", "NAME",
				"2048", "./code/gendata.C",
			},
		},
		"malformed filters": {
			args: []string{"-w", workflowName, "--filter", "name"},
			expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			wantError: true,
		},
		"no matching files:": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "du_no_files.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--filter",
				"name=nothing.C",
			},
			expected:  []string{"no files matching filter criteria"},
			wantError: true,
		},
		"unexisting workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{"-w", "invalid"},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist",
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
