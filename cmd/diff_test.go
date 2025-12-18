/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"reanahub/reana-client-go/pkg/datautils"
	"reanahub/reana-client-go/pkg/displayer"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

var diffPathTemplate = "/api/workflows/%s/diff/%s"

func TestDiff(t *testing.T) {
	workflowA := "my_workflow_a"
	workflowB := "my_workflow_b"

	tests := map[string]TestCmdParams{
		"all info": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					statusCode:   http.StatusOK,
					responseFile: "diff_complete.json",
				},
			},
			args: []string{workflowA, workflowB},
			expected: []string{
				"Differences in workflow version", "@@ -1 +1 @@", "- v0.1", "+ v0.2",
				"Differences in workflow inputs", "@@ -1 +2 @@", "- removed input", "+ added input", "+ more input",
				"Differences in workflow outputs", "@@ -2 +1 @@", "- removed output", "- more output", "+ added output",
				"Differences in workflow specification", "@@ +1 @@", "+ added specs",
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
		},
		"same specification": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					statusCode:   http.StatusOK,
					responseFile: "diff_same_spec.json",
				},
			},
			args: []string{workflowA, workflowB},
			expected: []string{
				"No differences in REANA specifications",
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
			unwanted: []string{
				"Differences in workflow version", "Differences in workflow inputs",
				"Differences in workflow specification", "Differences in workflow outputs",
			},
		},
		"no specification info": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					statusCode:   http.StatusOK,
					responseFile: "diff_no_spec.json",
				},
			},
			args: []string{workflowA, workflowB},
			expected: []string{
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
			unwanted: []string{
				"No differences in REANA specifications",
				"Differences in workflow version", "Differences in workflow inputs",
				"Differences in workflow specification", "Differences in workflow outputs",
			},
		},
		"no workspace info": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					statusCode:   http.StatusOK,
					responseFile: "diff_no_workspace.json",
				},
			},
			args: []string{workflowA, workflowB},
			expected: []string{
				"No differences in REANA specifications",
			},
			unwanted: []string{
				"Differences in workflow workspace",
			},
		},
		"unexisting workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{workflowA, workflowB},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
		"invalid number of arguments": {
			args:      []string{workflowA},
			expected:  []string{"accepts 2 arg(s), received 1"},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "diff"
			testCmdRun(t, params)
		})
	}
}

func TestPrintDiff(t *testing.T) {
	tests := map[string]struct {
		lines          []string
		expectedColors []text.Color
	}{
		"default text": {
			lines:          []string{"default text"},
			expectedColors: []text.Color{text.Reset},
		},
		"diff info": {
			lines:          []string{"@@ -1,14 +1,26 @@"},
			expectedColors: []text.Color{text.FgCyan},
		},
		"removed text": {
			lines:          []string{"- removed text"},
			expectedColors: []text.Color{text.FgRed},
		},
		"added text": {
			lines:          []string{"+ added text"},
			expectedColors: []text.Color{text.FgGreen},
		},
		"mixed text": {
			lines: []string{
				"@@ -1 +1 @@",
				"context",
				"- removed text",
				"+ added text",
			},
			expectedColors: []text.Color{
				text.FgCyan,
				text.Reset,
				text.FgRed,
				text.FgGreen,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resBuf := new(bytes.Buffer)
			printDiff(test.lines, resBuf)
			result := datautils.SplitLinesNoEmpty(resBuf.String())

			if len(result) != len(test.lines) {
				t.Fatalf(
					"Expected %d lines, got %d",
					len(test.lines),
					len(result),
				)
			}
			for i, line := range result {
				testBuf := new(bytes.Buffer)
				displayer.PrintColorable(
					test.lines[i],
					testBuf,
					test.expectedColors[i],
				)
				expected := testBuf.String()
				if line != expected {
					t.Errorf("Expected %s, got %s", expected, line)
				}
			}
		})
	}
}
