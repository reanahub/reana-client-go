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
	"reanahub/reana-client-go/utils"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

var diffPathTemplate = "/api/workflows/%s/diff/%s"

func TestDiff(t *testing.T) {
	workflowA := "my_workflow_a"
	workflowB := "my_workflow_b"
	diffResponse := `{
			"reana_specification":` + `"{` +
		`\"version\": [\"@@ -1 +1 @@\", \"- v0.1\", \"+ v0.2\"],` +
		`\"inputs\": [\"@@ -1 +2 @@\", \"- removed input\", \"+ added input\", \"+ more input\"],` +
		`\"outputs\": [\"@@ -2 +1 @@\", \"- removed output\", \"- more output\", \"+ added output\"],` +
		`\"workflow\": [\"@@ +1 @@\", \"+ added specs\"]` +
		`}"` + `,
			"workspace_listing": "\"Only in my_workflow_a: test.yaml\""
		}`
	sameSpecResponse := `{
			"reana_specification":` + `"{` +
		`\"version\": [],\"inputs\": [],\"outputs\": [],\"specification\": []` +
		`}"` + `,
			"workspace_listing": "\"Only in my_workflow_a: test.yaml\""
		}`
	noSpecResponse := `{
			"reana_specification": "",
			"workspace_listing": "\"Only in my_workflow_a: test.yaml\""
		}`
	noWorkspaceResponse := `{
			"reana_specification":` + `"{` +
		`\"version\": [],\"inputs\": [],\"outputs\": [],\"specification\": []` +
		`}"` + `,
			"workspace_listing": "\"\""
		}`

	tests := map[string]TestCmdParams{
		"all info": {
			serverResponse: diffResponse,
			statusCode:     http.StatusOK,
			args:           []string{workflowA, workflowB},
			expected: []string{
				"Differences in workflow version", "@@ -1 +1 @@", "- v0.1", "+ v0.2",
				"Differences in workflow inputs", "@@ -1 +2 @@", "- removed input", "+ added input", "+ more input",
				"Differences in workflow outputs", "@@ -2 +1 @@", "- removed output", "- more output", "+ added output",
				"Differences in workflow specification", "@@ +1 @@", "+ added specs",
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
		},
		"same specification": {
			serverResponse: sameSpecResponse,
			statusCode:     http.StatusOK,
			args:           []string{workflowA, workflowB},
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
			serverResponse: noSpecResponse,
			statusCode:     http.StatusOK,
			args:           []string{workflowA, workflowB},
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
			serverResponse: noWorkspaceResponse,
			statusCode:     http.StatusOK,
			args:           []string{workflowA, workflowB},
			expected: []string{
				"No differences in REANA specifications",
			},
			unwanted: []string{
				"Differences in workflow workspace",
			},
		},
		"unexisting workflow": {
			serverResponse: `{"message": "Workflow my_workflow_a does not exist."}`,
			statusCode:     http.StatusNotFound,
			args:           []string{workflowA, workflowB},
			expected:       []string{"Workflow my_workflow_a does not exist."},
			wantError:      true,
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
			params.serverPath = fmt.Sprintf(diffPathTemplate, workflowA, workflowB)
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
			lines:          []string{"@@ -1 +1 @@", "context", "- removed text", "+ added text"},
			expectedColors: []text.Color{text.FgCyan, text.Reset, text.FgRed, text.FgGreen},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resBuf := new(bytes.Buffer)
			printDiff(test.lines, resBuf)
			result := utils.SplitLinesNoEmpty(resBuf.String())

			if len(result) != len(test.lines) {
				t.Fatalf("Expected %d lines, got %d", len(test.lines), len(result))
			}
			for i, line := range result {
				testBuf := new(bytes.Buffer)
				utils.PrintColorable(test.lines[i], testBuf, test.expectedColors[i])
				expected := testBuf.String()
				if line != expected {
					t.Errorf("Expected %s, got %s", expected, line)
				}
			}
		})
	}
}
