/*
This file is part of REANA.
Copyright (C) 2023, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it under the terms
of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"testing"
)

var shareAddPathTemplate = "/api/workflows/%s/share"

func TestShareAdd(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{"-w", workflowName, "--user", "bob@cern.ch"},
			expected: []string{
				"my_workflow is now read-only shared with bob@cern.ch",
			},
		},
		"json": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{
				"-w", workflowName, "--user", "bob@cern.ch", "--json",
			},
			expected: []string{
				`"workflow": "my_workflow"`,
				`"shared_with": [`,
				`"bob@cern.ch"`,
				`"warnings": []`,
				`"errors": []`,
			},
			unwanted: []string{"SUCCESS"},
		},
		"email notification warning": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "share_add_email_warning.json",
				},
			},
			args: []string{"-w", workflowName, "--user", "bob@cern.ch"},
			expected: []string{
				"my_workflow is now read-only shared with bob@cern.ch",
				"bob@cern.ch: The workflow was shared, but the notification email could not be sent.",
			},
		},
		"email notification warning json": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "share_add_email_warning.json",
				},
			},
			args: []string{
				"-w", workflowName, "--user", "bob@cern.ch", "--json",
			},
			expected: []string{
				`"warnings": [`,
				"bob@cern.ch: The workflow was shared, but the notification email could not be sent.",
				`"errors": []`,
			},
		},
		"with message and valid-until": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{
				"-w", workflowName,
				"--user", "bob@cern.ch",
				"--message", "Please review my analysis",
				"--valid-until", "2099-12-31",
			},
			expected: []string{
				"my_workflow is now read-only shared with bob@cern.ch",
			},
		},
		"missing user": {
			args: []string{"-w", workflowName},
			expected: []string{
				"at least one of the options: 'user' is required",
			},
			wantError: true,
		},
		"invalid valid-until": {
			args: []string{
				"-w", workflowName,
				"--user", "bob@cern.ch",
				"--valid-until", "2024-02-30",
			},
			expected: []string{
				"invalid value for 'valid-until': '2024-02-30' does not match the format 'YYYY-MM-DD'",
			},
			wantError: true,
		},
		"invalid workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{
				"-w", "invalid",
				"--user", "bob@cern.ch",
			},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
		"invalid workflow json": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{
				"-w", "invalid",
				"--user", "bob@cern.ch",
				"--json",
			},
			expected: []string{
				`"workflow": "invalid"`,
				`"shared_with": []`,
				`"errors": [`,
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			unwanted:  []string{"ERROR:"},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "share-add"
			testCmdRun(t, params)
		})
	}
}

func TestValidateShareExpiry(t *testing.T) {
	tests := map[string]struct {
		value     string
		wantError bool
	}{
		"empty": {},
		"valid": {
			value: "2099-12-31",
		},
		"invalid calendar date": {
			value:     "2024-02-30",
			wantError: true,
		},
		"unpadded date": {
			value:     "2024-2-5",
			wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateShareExpiry(test.value)
			if test.wantError && err == nil {
				t.Fatal("expected an error")
			}
			if !test.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
