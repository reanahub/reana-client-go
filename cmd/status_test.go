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
	"reanahub/reana-client-go/client/operations"
	"testing"

	"github.com/go-gota/gota/series"

	"golang.org/x/exp/slices"
)

var statusPathTemplate = "/api/workflows/%s/status"

func TestStatus(t *testing.T) {
	workflowName := "my_workflow"
	successResponse := `{
		"created": "2022-07-20T12:08:40",
		"id": "my_workflow_id",
		"name": "my_workflow.10",
		"status": "finished",
		"user": "user",
		"logs": "logs",
		"progress": {
			"current_command": "ls",
			"current_step_name": "step_name",
			"failed": {
				"job_ids": [],
				"total": 0
			},
			"finished": {
				"job_ids": ["job1", "job2"],
				"total": 2
			},
			"run_finished_at": "2022-07-20T12:09:24",
			"run_started_at": "2022-07-20T12:09:09",
			"running": {
				"job_ids": ["job3", "job4"],
				"total": 2
			},
			"total": {
				"job_ids": [],
				"total": 2
			}
		}
	}`
	tests := map[string]TestCmdParams{
		"default": {
			serverPath:     fmt.Sprintf(statusPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName},
			expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS", "PROGRESS",
				workflowName, "10", "2022-07-20T12:08:40", "2022-07-20T12:09:09",
				"2022-07-20T12:09:24", "finished", "2/2",
			},
			unwanted: []string{"ID", "USER", "COMMAND", "DURATION"},
		},
		"format columns": {
			serverPath:     fmt.Sprintf(statusPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--format", "name,progress"},
			expected: []string{
				"NAME", "PROGRESS",
				workflowName, "2/2",
			},
			unwanted: []string{
				"RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
				"10", "2022-07-20T12:08:40", "2022-07-20T12:09:09",
				"2022-07-20T12:09:24", "finished",
			},
		},
		"invalid format column": {
			serverPath:     fmt.Sprintf(statusPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--format", "invalid"},
			expected: []string{
				"invalid value for 'format column': 'invalid' is not part of 'name', 'run_number', 'created', 'started', 'ended', 'status'",
			},
			wantError: true,
		},
		"json": {
			serverPath:     fmt.Sprintf(statusPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--json"},
			expected: []string{`[
  {
    "created": "2022-07-20T12:08:40",
    "ended": "2022-07-20T12:09:24",
    "name": "my_workflow",
    "progress": "2/2",
    "run_number": "10",
    "started": "2022-07-20T12:09:09",
    "status": "finished"
  }
]
`},
		},
		"verbose": {
			serverPath:     fmt.Sprintf(statusPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "-v"},
			expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED",
				"STATUS", "PROGRESS", "ID", "USER", "COMMAND", "DURATION",
				workflowName, "10", "2022-07-20T12:08:40", "2022-07-20T12:09:09",
				"2022-07-20T12:09:24", "finished", "2/2", "my_workflow_id",
				"user", "ls", "15",
			},
		},
		"include duration": {
			serverPath:     fmt.Sprintf(statusPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--include-duration"},
			expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "STARTED",
				"ENDED", "STATUS", "PROGRESS", "DURATION",
				workflowName, "10", "2022-07-20T12:08:40", "2022-07-20T12:09:09",
				"2022-07-20T12:09:24", "finished", "2/2", "15",
			},
			unwanted: []string{
				"ID", "USER", "COMMAND",
				"my_workflow_id", "user", "ls",
			},
		},
		"unexisting workflow": {
			serverPath:     fmt.Sprintf(statusPathTemplate, "invalid"),
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
			params.cmd = "status"
			testCmdRun(t, params)
		})
	}
}

func TestBuildStatusHeader(t *testing.T) {
	progressTotal := operations.GetWorkflowStatusOKBodyProgressTotal{Total: 60}

	tests := map[string]struct {
		verbose         bool
		includeDuration bool
		progress        operations.GetWorkflowStatusOKBodyProgress
		status          string
		expected        []string
	}{
		"created status": {
			status:   "created",
			expected: []string{"name", "run_number", "created", "status"},
		},
		"running without started info": {
			status:   "running",
			progress: operations.GetWorkflowStatusOKBodyProgress{RunFinishedAt: "dummy"},
			expected: []string{"name", "run_number", "created", "status"},
		},
		"running workflow": {
			status:   "running",
			progress: operations.GetWorkflowStatusOKBodyProgress{RunStartedAt: "dummy"},
			expected: []string{"name", "run_number", "created", "started", "status"},
		},
		"finished workflow": {
			status: "finished",
			progress: operations.GetWorkflowStatusOKBodyProgress{
				RunStartedAt:  "dummy",
				RunFinishedAt: "dummy",
			},
			expected: []string{"name", "run_number", "created", "started", "ended", "status"},
		},
		"with progress": {
			status:   "running",
			progress: operations.GetWorkflowStatusOKBodyProgress{Total: &progressTotal},
			expected: []string{"name", "run_number", "created", "status", "progress"},
		},
		"verbose": {
			status:   "running",
			verbose:  true,
			expected: []string{"name", "run_number", "created", "status", "id", "user", "duration"},
		},
		"verbose with command": {
			status:   "running",
			verbose:  true,
			progress: operations.GetWorkflowStatusOKBodyProgress{CurrentCommand: "cmd"},
			expected: []string{
				"name",
				"run_number",
				"created",
				"status",
				"id",
				"user",
				"command",
				"duration",
			},
		},
		"verbose with step": {
			status:   "running",
			verbose:  true,
			progress: operations.GetWorkflowStatusOKBodyProgress{CurrentStepName: "step"},
			expected: []string{
				"name",
				"run_number",
				"created",
				"status",
				"id",
				"user",
				"command",
				"duration",
			},
		},
		"include duration": {
			status:          "running",
			includeDuration: true,
			expected:        []string{"name", "run_number", "created", "status", "duration"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			header := buildStatusHeader(
				test.verbose,
				test.includeDuration,
				&test.progress,
				test.status,
			)
			if !slices.Equal(header, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, header)
			}
		})
	}
}

func TestGetStatusProgress(t *testing.T) {
	progressTotal := operations.GetWorkflowStatusOKBodyProgressTotal{Total: 60}
	progressFinished := operations.GetWorkflowStatusOKBodyProgressFinished{Total: 30}

	tests := map[string]struct {
		progress operations.GetWorkflowStatusOKBodyProgress
		expected string
	}{
		"no progress info": {
			expected: "-/-",
		},
		"with total": {
			progress: operations.GetWorkflowStatusOKBodyProgress{Total: &progressTotal},
			expected: "0/60",
		},
		"with finished": {
			progress: operations.GetWorkflowStatusOKBodyProgress{Finished: &progressFinished},
			expected: "-/-",
		},
		"with finished and total": {
			progress: operations.GetWorkflowStatusOKBodyProgress{
				Total:    &progressTotal,
				Finished: &progressFinished,
			},
			expected: "30/60",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			progress := getStatusProgress(&test.progress)
			if progress != test.expected {
				t.Errorf("expected %s, got %s", test.expected, progress)
			}
		})
	}
}

func TestGetStatusCommand(t *testing.T) {
	tests := map[string]struct {
		progress operations.GetWorkflowStatusOKBodyProgress
		expected string
	}{
		"no command": {
			progress: operations.GetWorkflowStatusOKBodyProgress{CurrentStepName: "step"},
			expected: "step",
		},
		"with command": {
			progress: operations.GetWorkflowStatusOKBodyProgress{
				CurrentCommand:  "cmd",
				CurrentStepName: "step",
			},
			expected: "cmd",
		},
		"command with prefix": {
			progress: operations.GetWorkflowStatusOKBodyProgress{
				CurrentCommand:  "bash -c \"cd folder; ls \"",
				CurrentStepName: "step",
			},
			expected: "ls",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			command := getStatusCommand(&test.progress)
			if command != test.expected {
				t.Errorf("expected %s, got %s", test.expected, command)
			}
		})
	}
}

func TestBuildStatusSeries(t *testing.T) {
	tests := map[string]struct {
		col      string
		expected series.Series
	}{
		"regular column": {
			col: "name", expected: series.New([]string{}, series.String, "name"),
		},
		"duration": {
			col: "duration", expected: series.New([]int{}, series.Int, "duration"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildStatusSeries(test.col)
			if got.Name != test.expected.Name {
				t.Errorf("series has name '%s', wanted '%s'", got.Name, test.expected.Name)
			}
			if got.Type() != test.expected.Type() {
				t.Errorf("series has type '%s', wanted '%s'", got.Type(), test.expected.Type())
			}
		})
	}
}
