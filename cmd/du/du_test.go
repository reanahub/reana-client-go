/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package du

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var duPathTemplate = "/api/workflows/%s/disk_usage"

func TestDu(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"SIZE", "NAME",
				"2048", "./code/fitdata.C",
				"4608", "./code/gendata.C",
			},
		},
		"summarize": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/summarize.json",
				},
			},
			Args: []string{"-w", workflowName, "-s"},
			Expected: []string{
				"SIZE", "NAME",
				"10240", ".",
			},
		},
		"human readable": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "-h"},
			Expected: []string{
				"SIZE", "NAME",
				"2 KiB", "./code/fitdata.C",
				"4.5 KiB", "./code/gendata.C",
			},
		},
		"files in black list": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/blacklist.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"SIZE", "NAME",
				"4608", "./code/gendata.C",
			},
			Unwanted: []string{
				"2048", "./git/test.C",
			},
		},
		"with filters": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/filters.json",
				},
			},
			Args: []string{"-w", workflowName, "--filter", "name=./code/gendata.C,size=2048"},
			Expected: []string{
				"SIZE", "NAME",
				"2048", "./code/gendata.C",
			},
		},
		"malformed filters": {
			Args: []string{"-w", workflowName, "--filter", "name"},
			Expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			WantError: true,
		},
		"no matching files:": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_files.json",
				},
			},
			Args:      []string{"-w", workflowName, "--filter", "name=nothing.C"},
			Expected:  []string{"no files matching filter criteria"},
			WantError: true,
		},
		"unexisting workflow": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(duPathTemplate, "invalid"): {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "../../testdata/inputs/invalid_workflow.json",
				},
			},
			Args: []string{"-w", "invalid"},
			Expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist",
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
