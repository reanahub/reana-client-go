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
	"reanahub/reana-client-go/pkg/filterer"
	"reflect"
	"testing"

	"golang.org/x/exp/slices"
)

var logsPathTemplate = "/api/workflows/%s/logs"

func TestLogs(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"Workflow engine logs", "workflow logs",
				"Engine internal logs", "engine logs",
				"Job logs", "Step:", "job1", "Workflow ID:", "workflow_1",
				"Compute backend:", "Kubernetes", "Job ID:", "backend1",
				"Docker image:", "docker1", "Command:", "ls", "Status:", "finished",
				"Started:", "2022-07-20T12:09:09", "Finished:", "2022-07-20T19:09:09",
				"Logs:", "workflow 1 logs", "Step:", "job2",
			},
		},
		"without log information": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_empty.json",
				},
			},
			args: []string{"-w", workflowName},
			unwanted: []string{
				"Workflow engine logs", "Engine internal logs",
				"Job logs", "Step:", "job1",
			},
		},
		"json": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete.json",
				},
			},
			args: []string{"-w", workflowName, "--json"},
			expected: []string{
				"\"workflow_logs\": \"workflow logs\"",
				"\"job_logs\": {", "\"1\": {",
				"\"workflow_uuid\": \"workflow_1\"",
				"\"logs\": \"workflow 1 logs\"",
				"\"engine_specific\": \"engine logs\"",
			},
		},
		"with filters": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--filter",
				"compute_backend=kubernetes",
			},
			expected: []string{"Step: job1"},
			unwanted: []string{"Step: job2"},
		},
		"missing step names": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete.json",
				},
			},
			args: []string{"-w", workflowName, "--filter", "step=3"},
			expected: []string{
				"ERROR:", "The logs of step(s) 3 were not found, check for spelling mistakes in the step names",
			},
		},
		"missing fields": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_incomplete.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--filter",
				"compute_backend=kubernetes",
			},
			expected: []string{"Step: 1", "Step 1 emitted no logs."},
			unwanted: []string{
				"job1",
				"Workflow ID:", "workflow_1",
				"Logs:", "workflow 1 logs",
			},
		},
		"malformed filters": {
			args: []string{"-w", workflowName, "--filter", "name"},
			expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			wantError: true,
		},
		"unexisting workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{"-w", "invalid"},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
		"invalid page": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusBadRequest,
					responseFile: "common_invalid_page.json",
				},
			},
			args:      []string{"-w", workflowName, "--page", "0"},
			expected:  []string{"Field 'page': Must be at least 1."},
			wantError: true,
		},
		"invalid server url": {
			serverURL: "^^*invalid",
			args:      []string{"-w", workflowName},
			expected: []string{
				"environment variable REANA_SERVER_URL is not set",
			},
			wantError: true,
		},
		"follow workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete_live.json",
				},
				fmt.Sprintf(statusPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "status_finished.json",
				},
			},
			args: []string{"-w", workflowName, "--follow", "-i", "0"},
			expected: []string{
				"Interval must be greater than or equal to 1, using default interval (10 s).",
				"workflow logs",
				"==> Workflow has completed, you might want to rerun the command without the --follow flag.",
			},
			unwanted: []string{
				"job1",
				"step",
			},
		},
		"follow job with multiple steps, size, interval and json flags": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_running.json",
					additionalResponseFiles: []string{
						"logs_complete_live.json",
					},
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--follow",
				"--json",
				"--filter",
				"step=job1",
				"--filter",
				"step=job2",
				"--size",
				"1",
				"-i",
				"1",
			},
			expected: []string{
				"Ignoring --json as it cannot be used together with --follow.",
				"Only one step can be followed at a time, ignoring additional steps.",
				"workflow 1 logs",
				"Job has completed, you might want to rerun the command without the --follow flag.",
			},
		},
		"follow job that does not exist": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_empty.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--follow",
				"--filter",
				"step=job1",
			},
			expected: []string{
				"step job1 not found",
			},
			wantError: true,
		},
		"follow logs when server returns logs error": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode: http.StatusInternalServerError,
				},
			},
			args:      []string{"-w", workflowName, "--follow"},
			wantError: true,
		},
		"follow logs when server returns status error": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete_live.json",
				},
				fmt.Sprintf(statusPathTemplate, workflowName): {
					statusCode: http.StatusInternalServerError,
				},
			},
			args:      []string{"-w", workflowName, "--follow"},
			wantError: true,
		},
		"follow logs when server returns html response": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "page.html",
				},
			},
			args: []string{"-w", workflowName, "--follow"},
			expected: []string{
				"invalid character '<' looking for beginning of value",
			},
			wantError: true,
		},
		"follow logs when server returns malformed logs": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_malformed.json",
				},
			},
			args: []string{"-w", workflowName, "--follow"},
			expected: []string{
				"invalid character 'm' looking for beginning of value",
			},
			wantError: true,
		},
		"follow logs when live logs are disabled": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "logs_complete.json",
				},
			},
			args: []string{"-w", workflowName, "--follow"},
			expected: []string{
				"live logs are not enabled, please rerun the command without the --follow flag",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "logs"
			testCmdRun(t, params)
		})
	}
}

func TestParseLogsFilters(t *testing.T) {
	tests := map[string]struct {
		filterInput []string
		wantError   bool
	}{
		"valid filters": {
			filterInput: []string{
				"compute_backend=kubernetes",
				"status=running",
				"docker_img=docker",
			},
		},
		"invalid filter key": {
			filterInput: []string{"invalid=kubernetes"},
			wantError:   true,
		},
		"invalid status filter": {
			filterInput: []string{"status=invalid"},
			wantError:   true,
		},
		"invalid compute backend filter": {
			filterInput: []string{"compute_backend=invalid"},
			wantError:   true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseLogsFilters(test.filterInput)
			if test.wantError && err == nil {
				t.Fatalf(
					"expected parseLogsFilters(%#v) to return an error but didn't",
					test.filterInput,
				)
			}
			if !test.wantError && err != nil {
				t.Fatalf(
					"parseLogsFilters(%#v) returned an unexpected error: %s",
					test.filterInput,
					err.Error(),
				)
			}
		})
	}

	t.Run("expected filter keys", func(t *testing.T) {
		filters, err := parseLogsFilters([]string{})
		if err != nil {
			t.Fatalf(
				"parseLogsFilters(%#v) returned an unexpected error: %s",
				[]string{},
				err.Error(),
			)
		}
		if !slices.Equal(filters.SingleFilterKeys, config.LogsSingleFilters) {
			t.Fatalf(
				"expected single filter keys to be %#v but got %#v",
				config.LogsSingleFilters,
				filters.SingleFilterKeys,
			)
		}
		if !slices.Equal(filters.MultiFilterKeys, config.LogsMultiFilters) {
			t.Fatalf(
				"expected multi filter keys to be %#v but got %#v",
				config.LogsMultiFilters,
				filters.MultiFilterKeys,
			)
		}
	})
}

func TestFilterJobLogs(t *testing.T) {
	tests := map[string]struct {
		filterInput []string
		wantLogs    map[string]jobLogItem
	}{
		"no filters": {
			filterInput: []string{},
			wantLogs: map[string]jobLogItem{
				"1": {
					ComputeBackend: "Kubernetes",
					Status:         "running",
					DockerImg:      "docker",
				},
				"2": {
					ComputeBackend: "Slurm",
					Status:         "created",
					DockerImg:      "docker2",
				},
				"3": {
					ComputeBackend: "HTCondor",
					Status:         "created",
					DockerImg:      "docker3",
				},
			},
		},
		"single filter": {
			filterInput: []string{"status=created"},
			wantLogs: map[string]jobLogItem{
				"2": {
					ComputeBackend: "Slurm",
					Status:         "created",
					DockerImg:      "docker2",
				},
				"3": {
					ComputeBackend: "HTCondor",
					Status:         "created",
					DockerImg:      "docker3",
				},
			},
		},
		"multiple filters": {
			filterInput: []string{
				"status=created",
				"compute_backend=slurm",
				"docker_img=docker2",
			},
			wantLogs: map[string]jobLogItem{
				"2": {
					ComputeBackend: "Slurm",
					Status:         "created",
					DockerImg:      "docker2",
				},
			},
		},
		"uppercase compute_backend": {
			filterInput: []string{"compute_backend=KUBERNETES"},
			wantLogs: map[string]jobLogItem{
				"1": {
					ComputeBackend: "Kubernetes",
					Status:         "running",
					DockerImg:      "docker",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := filterer.NewFilters(
				config.LogsSingleFilters,
				config.LogsMultiFilters,
				test.filterInput,
			)
			if err != nil {
				t.Fatalf(
					"utils.NewFilters returned an unexpected error: %s",
					err.Error(),
				)
			}

			jobLogs := map[string]jobLogItem{
				"1": {
					ComputeBackend: "Kubernetes",
					Status:         "running",
					DockerImg:      "docker",
				},
				"2": {
					ComputeBackend: "Slurm",
					Status:         "created",
					DockerImg:      "docker2",
				},
				"3": {
					ComputeBackend: "HTCondor",
					Status:         "created",
					DockerImg:      "docker3",
				},
			}
			err = filterJobLogs(&jobLogs, filters)
			if err != nil {
				t.Fatalf(
					"filterJobLogs returned an unexpected error: %s",
					err.Error(),
				)
			}
			if !reflect.DeepEqual(jobLogs, test.wantLogs) {
				t.Errorf("expected %#v, got %#v", test.wantLogs, jobLogs)
			}
		})
	}
}
