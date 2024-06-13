/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"reanahub/reana-client-go/pkg/config"
	"testing"

	"github.com/go-gota/gota/series"

	"golang.org/x/exp/slices"
)

var listServerPath = "/api/workflows"

func TestList(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
				"my_workflow", "23", "2022-07-28T12:04:37", "2022-07-28T12:04:52",
				"2022-07-28T12:13:10", "finished", "my_workflow2", "12",
				"2022-08-10T17:14:12", "2022-08-10T18:04:52", "-", "running",
			},
			unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
			},
		},
		"interactive sessions": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"-s"},
			expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
				"my_workflow", "23", "2022-07-28T12:04:37", "jupyter", "/session1uri", "created",
				"my_workflow2", "12", "2022-08-10T17:14:12", "/session2uri",
			},
			unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"STARTED", "ENDED", " STATUS",
			},
		},
		"format columns": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--format", "name,status"},
			expected: []string{
				"NAME", "STATUS",
				"my_workflow", "finished",
				"my_workflow2", "running",
			},
			unwanted: []string{
				"RUN_NUMBER", "CREATED", "STARTED", "ENDED",
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
			},
		},
		"format with filter": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--format", "name=my_workflow,status"},
			expected: []string{
				"NAME", "STATUS",
				"my_workflow", "finished",
			},
			unwanted: []string{
				"RUN_NUMBER", "CREATED", "STARTED", "ENDED",
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
				"my_workflow2", "running",
			},
		},
		"invalid format column": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--format", "invalid"},
			expected: []string{
				"invalid value for 'format column': 'invalid' is not part of 'name', 'run_number', 'created', 'started', 'ended', 'status'",
			},
			wantError: true,
		},
		"json": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--json"},
			expected: []string{`[
  {
    "created": "2022-08-10T17:14:12",
    "ended": null,
    "name": "my_workflow2",
    "run_number": "12",
    "started": "2022-08-10T18:04:52",
    "status": "running"
  },
  {
    "created": "2022-07-28T12:04:37",
    "ended": "2022-07-28T12:13:10",
    "name": "my_workflow",
    "run_number": "23",
    "started": "2022-07-28T12:04:52",
    "status": "finished"
  }
]
`},
		},
		"verbose": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"-v"},
			expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"my_workflow", "23", "2022-07-28T12:04:37", "2022-07-28T12:04:52",
				"2022-07-28T12:13:10", "finished", "my_workflow_id", "user",
				"1024", "2/2", "498",
				"my_workflow2", "12", "2022-08-10T17:14:12",
				"2022-08-10T18:04:52", "-", "running", "my_workflow2_id",
				" -1 ", "1/2",
			},
		},
		"raw size": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args:     []string{"--include-workspace-size"},
			expected: []string{"SIZE", "1024", " -1 "},
			unwanted: []string{"1 KiB"},
		},
		"human readable size": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args:     []string{"--include-workspace-size", "-h"},
			expected: []string{"SIZE", "1 KiB"},
			unwanted: []string{"1024", " -1 "},
		},
		"include duration": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args:     []string{"--include-duration"},
			expected: []string{"DURATION", "498"},
			unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "SESSION_TYPE",
				"SESSION_URI", "SESSION_STATUS",
			},
		},
		"include progress": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args:     []string{"--include-progress"},
			expected: []string{"PROGRESS", "2/2", "1/2"},
			unwanted: []string{
				"ID", "USER", "SIZE", "DURATION", "SESSION_TYPE",
				"SESSION_URI", "SESSION_STATUS",
			},
		},
		"sorted": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args:     []string{"--sort", "run_number"},
			expected: []string{"STATUS  \nmy_workflow "},
		},
		"malformed filters": {
			args: []string{"--filter", "name"},
			expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			wantError: true,
		},
		"unexisting workflow": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
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
		"invalid size": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusBadRequest,
					responseFile: "common_invalid_size.json",
				},
			},
			args:      []string{"--size", "0"},
			expected:  []string{"Field 'size': Must be at least 1."},
			wantError: true,
		},
		"invalid sort columns": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--sort", "invalid"},
			expected: []string{
				"Warning: sort operation was aborted, column 'invalid' does not exist",
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
			},
		},
		"include shared by others": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--shared"},
			expected: []string{
				"SHARED_BY", "SHARED_WITH",
			},
		},
		"list shared with user": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--shared-with", "anybody"},
			expected: []string{
				"SHARED_WITH",
			},
			unwanted: []string{
				"SHARED_BY",
			},
		},
		"list shared by user": {
			serverResponses: map[string]ServerResponse{
				listServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "list.json",
				},
			},
			args: []string{"--shared-by", "anybody"},
			expected: []string{
				"SHARED_BY",
			},
			unwanted: []string{
				"SHARED_WITH",
			},
		},
		"invalid: shared with and shared by in the same command": {
			args: []string{"--shared-by", "anybody", "--shared-with", "anybody"},
			expected: []string{
				"Please provide either --shared-by or --shared-with, not both",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "list"
			testCmdRun(t, params)
		})
	}
}

func TestBuildListHeader(t *testing.T) {
	tests := map[string]struct {
		runType              string
		verbose              bool
		includeWorkspaceSize bool
		includeProgress      bool
		includeDuration      bool
		shared               bool
		shared_by            string
		shared_with          string
		expected             []string
	}{
		"batch run": {
			runType:  "batch",
			expected: []string{"name", "run_number", "created", "started", "ended", "status"},
		},
		"interactive run": {
			runType: "interactive",
			expected: []string{
				"name", "run_number", "created",
				"session_type", "session_uri", "session_status",
			},
		},
		"verbose": {
			runType: "batch",
			verbose: true,
			expected: []string{
				"name", "run_number", "created", "started", "ended",
				"status", "id", "user", "size", "progress", "duration",
			},
		},
		"include workspace size": {
			runType:              "batch",
			includeWorkspaceSize: true,
			expected: []string{
				"name", "run_number", "created", "started",
				"ended", "status", "size",
			},
		},
		"include progress": {
			runType:         "batch",
			includeProgress: true,
			expected: []string{
				"name", "run_number", "created", "started",
				"ended", "status", "progress",
			},
		},
		"include duration": {
			runType:         "batch",
			includeDuration: true,
			expected: []string{
				"name", "run_number", "created", "started",
				"ended", "status", "duration",
			},
		},
		"shared": {
			runType: "batch",
			shared:  true,
			expected: []string{
				"name", "run_number", "created", "started",
				"ended", "status", "shared_with", "shared_by",
			},
		},
		"shared by": {
			runType:   "batch",
			shared_by: "user@example.org",
			expected: []string{
				"name", "run_number", "created", "started",
				"ended", "status", "shared_by",
			},
		},
		"shared with": {
			runType:     "batch",
			shared_with: "user@example.org",
			expected: []string{
				"name", "run_number", "created", "started",
				"ended", "status", "shared_with",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			header := buildListHeader(
				test.runType,
				test.verbose,
				test.includeWorkspaceSize,
				test.includeProgress,
				test.includeDuration,
				test.shared,
				test.shared_by,
				test.shared_with,
			)
			if !slices.Equal(header, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, header)
			}
		})
	}
}

func TestParseListFilters(t *testing.T) {
	tests := map[string]struct {
		filterInput     []string
		showDeletedRuns bool
		showAll         bool
		statusFilters   []string
		searchFilter    string
		wantError       bool
	}{
		"no filters": {
			filterInput:   []string{},
			statusFilters: config.GetRunStatuses(false),
			searchFilter:  "",
		},
		"with deleted runs": {
			filterInput:     []string{},
			showDeletedRuns: true,
			statusFilters:   config.GetRunStatuses(true),
			searchFilter:    "",
		},
		"with show all": {
			filterInput:   []string{},
			showAll:       true,
			statusFilters: config.GetRunStatuses(true),
			searchFilter:  "",
		},
		"valid filters": {
			filterInput:   []string{"status=running", "status=finished", "name=test", "name=test2"},
			statusFilters: []string{"running", "finished"},
			searchFilter:  "{\"name\":[\"test\",\"test2\"]}",
		},
		"invalid filter key": {
			filterInput: []string{"key=value"},
			wantError:   true,
		},
		"invalid status filter": {
			filterInput: []string{"status=invalid"},
			wantError:   true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			statusFilters, searchFilter, err := parseListFilters(
				test.filterInput, test.showDeletedRuns, test.showAll,
			)
			if test.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				if !slices.Equal(statusFilters, test.statusFilters) {
					t.Errorf("expected status filters to be %v, got %v", test.statusFilters, statusFilters)
				}
				if searchFilter != test.searchFilter {
					t.Errorf("expected search filter to be %s, got %s", test.searchFilter, searchFilter)
				}
			}
		})
	}
}

func TestBuildListSeries(t *testing.T) {
	tests := map[string]struct {
		col           string
		humanReadable bool
		expected      series.Series
	}{
		"regular column": {
			col: "name", expected: series.New([]string{}, series.String, "name"),
		},
		"duration": {
			col: "duration", expected: series.New([]int{}, series.Int, "duration"),
		},
		"raw size": {
			col: "size", expected: series.New([]int{}, series.Int, "size"),
		},
		"human readable size": {
			col: "size", humanReadable: true, expected: series.New([]string{}, series.String, "size"),
		},
		"human readable other column": {
			col: "name", humanReadable: true, expected: series.New([]string{}, series.String, "name"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildListSeries(test.col, test.humanReadable)
			if got.Name != test.expected.Name {
				t.Errorf("series has name '%s', wanted '%s'", got.Name, test.expected.Name)
			}
			if got.Type() != test.expected.Type() {
				t.Errorf("series has type '%s', wanted '%s'", got.Type(), test.expected.Type())
			}
		})
	}
}

func TestGetOptionalStringField(t *testing.T) {
	emptyString := ""
	validString := "valid"

	tests := map[string]struct {
		value    *string
		expected any
	}{
		"empty string": {
			value:    &emptyString,
			expected: nil,
		},
		"nil string": {
			value:    nil,
			expected: nil,
		},
		"valid string": {
			value:    &validString,
			expected: validString,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := getOptionalStringField(test.value)
			if got != test.expected {
				t.Errorf("got %v, expected %v", got, test.expected)
			}
		})
	}
}

func TestGetProgressField(t *testing.T) {
	tests := map[string]struct {
		value    int64
		expected string
	}{
		"zero value": {
			value:    0,
			expected: "-",
		},
		"non zero value": {
			value:    100,
			expected: "100",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := getProgressField(test.value)
			if got != test.expected {
				t.Errorf("got %s, expected %s", got, test.expected)
			}
		})
	}
}
