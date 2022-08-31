/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package logs

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
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
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
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
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/empty.json",
				},
			},
			Args: []string{"-w", workflowName},
			Unwanted: []string{
				"Workflow engine logs", "Engine internal logs",
				"Job logs", "Step:", "job1",
			},
		},
		"json": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--json"},
			Expected: []string{
				"\"workflow_logs\": \"workflow logs\"",
				"\"job_logs\": {", "\"1\": {",
				"\"workflow_uuid\": \"workflow_1\"",
				"\"logs\": \"workflow 1 logs\"",
				"\"engine_specific\": \"engine logs\"",
			},
		},
		"with filters": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"-w", workflowName, "--filter", "compute_backend=kubernetes"},
			Expected: []string{"Step: job1"},
			Unwanted: []string{"Step: job2"},
		},
		"missing step names": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--filter", "step=3"},
			Expected: []string{
				"ERROR:", "The logs of step(s) 3 were not found, check for spelling mistakes in the step names",
			},
		},
		"missing fields": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/incomplete.json",
				},
			},
			Args:     []string{"-w", workflowName, "--filter", "compute_backend=kubernetes"},
			Expected: []string{"Step: 1", "Step 1 emitted no logs."},
			Unwanted: []string{
				"job1",
				"Workflow ID:", "workflow_1",
				"Logs:", "workflow 1 logs",
			},
		},
		"malformed filters": {
			Args: []string{"-w", workflowName, "--filter", "name"},
			Expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			WantError: true,
		},
		"unexisting workflow": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, "invalid"): {
					StatusCode:   http.StatusNotFound,
					ResponseFile: "../../testdata/inputs/invalid_workflow.json",
				},
			},
			Args: []string{"-w", "invalid"},
			Expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			WantError: true,
		},
		"invalid page": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(logsPathTemplate, workflowName): {
					StatusCode:   http.StatusBadRequest,
					ResponseFile: "../../testdata/inputs/invalid_page.json",
				},
			},
			Args:      []string{"-w", workflowName, "--page", "0"},
			Expected:  []string{"Field 'page': Must be at least 1."},
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
			_, err := parseFilters(test.filterInput)
			if test.wantError && err == nil {
				t.Fatalf(
					"expected parseFilters(%#v) to return an error but didn't",
					test.filterInput,
				)
			}
			if !test.wantError && err != nil {
				t.Fatalf(
					"parseFilters(%#v) returned an unexpected error: %s",
					test.filterInput,
					err.Error(),
				)
			}
		})
	}

	t.Run("expected filter keys", func(t *testing.T) {
		filters, err := parseFilters([]string{})
		if err != nil {
			t.Fatalf(
				"parseFilters(%#v) returned an unexpected error: %s",
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
				"1": {ComputeBackend: "Kubernetes", Status: "running", DockerImg: "docker"},
				"2": {ComputeBackend: "Slurm", Status: "created", DockerImg: "docker2"},
				"3": {ComputeBackend: "HTCondor", Status: "created", DockerImg: "docker3"},
			},
		},
		"single filter": {
			filterInput: []string{"status=created"},
			wantLogs: map[string]jobLogItem{
				"2": {ComputeBackend: "Slurm", Status: "created", DockerImg: "docker2"},
				"3": {ComputeBackend: "HTCondor", Status: "created", DockerImg: "docker3"},
			},
		},
		"multiple filters": {
			filterInput: []string{"status=created", "compute_backend=slurm", "docker_img=docker2"},
			wantLogs: map[string]jobLogItem{
				"2": {ComputeBackend: "Slurm", Status: "created", DockerImg: "docker2"},
			},
		},
		"uppercase compute_backend": {
			filterInput: []string{"compute_backend=KUBERNETES"},
			wantLogs: map[string]jobLogItem{
				"1": {ComputeBackend: "Kubernetes", Status: "running", DockerImg: "docker"},
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
				t.Fatalf("utils.NewFilters returned an unexpected error: %s", err.Error())
			}

			jobLogs := map[string]jobLogItem{
				"1": {ComputeBackend: "Kubernetes", Status: "running", DockerImg: "docker"},
				"2": {ComputeBackend: "Slurm", Status: "created", DockerImg: "docker2"},
				"3": {ComputeBackend: "HTCondor", Status: "created", DockerImg: "docker3"},
			}
			err = filterJobLogs(&jobLogs, filters)
			if err != nil {
				t.Fatalf("filterJobLogs returned an unexpected error: %s", err.Error())
			}
			if !reflect.DeepEqual(jobLogs, test.wantLogs) {
				t.Errorf("expected %#v, got %#v", test.wantLogs, jobLogs)
			}
		})
	}
}
