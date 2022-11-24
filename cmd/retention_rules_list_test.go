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

var retentionRulesPathTemplate = "/api/workflows/%s/retention_rules"

func TestRetentionRulesList(t *testing.T) {
	workflowName := "my_workflow"

	tests := map[string]TestCmdParams{
		"missing workflow": {
			wantError: true,
			expected:  []string{"--workflow"},
		},
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(retentionRulesPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "retention_rules_active.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"WORKSPACE_FILES", "RETENTION_DAYS", "APPLY_ON", "STATUS",
				"*.csv", " 1 ", "2022-11-25T23:59:59", "active",
				"**/*.txt", " 2 ", "2022-11-26T23:59:59", "active",
			},
		},
		"apply_on null": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(retentionRulesPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "retention_rules_apply_on_null.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"WORKSPACE_FILES", "RETENTION_DAYS", "APPLY_ON", "STATUS",
				"**/*.txt", "1", "-", "created",
			},
			unwanted: []string{"nil", "null"},
		},
		"invalid workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(retentionRulesPathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args:      []string{"-w", "invalid"},
			wantError: true,
			expected:  []string{"REANA_WORKON", "invalid"},
		},
		"apply_on null json output": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(retentionRulesPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "retention_rules_apply_on_null.json",
				},
			},
			args:     []string{"-w", workflowName, "--json"},
			expected: []string{"\"apply_on\": null,"},
			unwanted: []string{"-"},
		},
		"format filters": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(retentionRulesPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "retention_rules_active.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--format",
				"status=active",
				"--format",
				"workspace_files",
			},
			expected: []string{
				"WORKSPACE_FILES", "STATUS",
				"*.csv", "active",
				"**/*.txt", "active",
			},
			unwanted: []string{
				"RETENTION_DAYS", "APPLY_ON",
				" 1 ", "2022-11-25T23:59:59",
				" 2 ", "2022-11-26T23:59:59",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "retention-rules-list"
			testCmdRun(t, params)
		})
	}
}
