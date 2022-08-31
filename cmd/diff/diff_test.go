/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package diff

import (
	"bytes"
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
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
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{workflowA, workflowB},
			Expected: []string{
				"Differences in workflow version", "@@ -1 +1 @@", "- v0.1", "+ v0.2",
				"Differences in workflow inputs", "@@ -1 +2 @@", "- removed input", "+ added input", "+ more input",
				"Differences in workflow outputs", "@@ -2 +1 @@", "- removed output", "- more output", "+ added output",
				"Differences in workflow specification", "@@ +1 @@", "+ added specs",
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
		},
		"same specification": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/same_spec.json",
				},
			},
			Args: []string{workflowA, workflowB},
			Expected: []string{
				"No differences in REANA specifications",
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
			Unwanted: []string{
				"Differences in workflow version", "Differences in workflow inputs",
				"Differences in workflow specification", "Differences in workflow outputs",
			},
		},
		"no specification info": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_spec.json",
				},
			},
			Args: []string{workflowA, workflowB},
			Expected: []string{
				"Differences in workflow workspace", "Only in my_workflow_a: test.yaml",
			},
			Unwanted: []string{
				"No differences in REANA specifications",
				"Differences in workflow version", "Differences in workflow inputs",
				"Differences in workflow specification", "Differences in workflow outputs",
			},
		},
		"no workspace info": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_workspace.json",
				},
			},
			Args: []string{workflowA, workflowB},
			Expected: []string{
				"No differences in REANA specifications",
			},
			Unwanted: []string{
				"Differences in workflow workspace",
			},
		},
		"unexisting workflow": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(diffPathTemplate, workflowA, workflowB): {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "../../testdata/inputs/invalid_workflow.json",
				},
			},
			Args: []string{workflowA, workflowB},
			Expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			WantError: true,
		},
		"invalid number of arguments": {
			Args:      []string{workflowA},
			Expected:  []string{"accepts 2 arg(s), received 1"},
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
			result := datautils.SplitLinesNoEmpty(resBuf.String())

			if len(result) != len(test.lines) {
				t.Fatalf("Expected %d lines, got %d", len(test.lines), len(result))
			}
			for i, line := range result {
				testBuf := new(bytes.Buffer)
				displayer.PrintColorable(test.lines[i], testBuf, test.expectedColors[i])
				expected := testBuf.String()
				if line != expected {
					t.Errorf("Expected %s, got %s", expected, line)
				}
			}
		})
	}
}
