/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package list

import (
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"reanahub/reana-client-go/pkg/config"
	"testing"

	"github.com/go-gota/gota/series"

	"golang.org/x/exp/slices"
)

var listServerPath = "/api/workflows"

func TestList(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
				"my_workflow", "23", "2022-07-28T12:04:37", "2022-07-28T12:04:52",
				"2022-07-28T12:13:10", "finished", "my_workflow2", "12",
				"2022-08-10T17:14:12", "2022-08-10T18:04:52", "-", "running",
			},
			Unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
			},
		},
		"interactive sessions": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-s"},
			Expected: []string{
				"NAME", "RUN_NUMBER", "CREATED", "SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
				"my_workflow", "23", "2022-07-28T12:04:37", "jupyter", "/session1uri", "created",
				"my_workflow2", "12", "2022-08-10T17:14:12", "/session2uri",
			},
			Unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"STARTED", "ENDED", " STATUS",
			},
		},
		"format columns": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"--format", "name,status"},
			Expected: []string{
				"NAME", "STATUS",
				"my_workflow", "finished",
				"my_workflow2", "running",
			},
			Unwanted: []string{
				"RUN_NUMBER", "CREATED", "STARTED", "ENDED",
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
			},
		},
		"format with filter": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"--format", "name=my_workflow,status"},
			Expected: []string{
				"NAME", "STATUS",
				"my_workflow", "finished",
			},
			Unwanted: []string{
				"RUN_NUMBER", "CREATED", "STARTED", "ENDED",
				"ID", "USER", "SIZE", "PROGRESS", "DURATION",
				"SESSION_TYPE", "SESSION_URI", "SESSION_STATUS",
				"my_workflow2", "running",
			},
		},
		"invalid format column": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"--format", "invalid"},
			Expected: []string{
				"invalid value for 'format column': 'invalid' is not part of 'name', 'run_number', 'created', 'started', 'ended', 'status'",
			},
			WantError: true,
		},
		"json": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"--json"},
			Expected: []string{`[
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
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-v"},
			Expected: []string{
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
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--include-workspace-size"},
			Expected: []string{"SIZE", "1024", " -1 "},
			Unwanted: []string{"1 KiB"},
		},
		"human readable size": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--include-workspace-size", "-h"},
			Expected: []string{"SIZE", "1 KiB"},
			Unwanted: []string{"1024", " -1 "},
		},
		"include duration": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--include-duration"},
			Expected: []string{"DURATION", "498"},
			Unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "SESSION_TYPE",
				"SESSION_URI", "SESSION_STATUS",
			},
		},
		"include progress": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--include-progress"},
			Expected: []string{"PROGRESS", "2/2", "1/2"},
			Unwanted: []string{
				"ID", "USER", "SIZE", "DURATION", "SESSION_TYPE",
				"SESSION_URI", "SESSION_STATUS",
			},
		},
		"sorted": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--sort", "run_number"},
			Expected: []string{"STATUS  \nmy_workflow "},
		},
		"malformed filters": {
			Args: []string{"--filter", "name"},
			Expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			WantError: true,
		},
		"unexisting workflow": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
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
		"invalid size": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusBadRequest,
					ResponseFile: "../../testdata/inputs/invalid_size.json",
				},
			},
			Args:      []string{"--size", "0"},
			Expected:  []string{"Field 'size': Must be at least 1."},
			WantError: true,
		},
		"invalid sort columns": {
			ServerResponses: map[string]ServerResponse{
				listServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"--sort", "invalid"},
			Expected: []string{
				"Warning: sort operation was aborted, column 'invalid' does not exist",
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.Cmd = NewCmd()
			TestCmdRun(t, params)
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			header := buildHeader(
				test.runType,
				test.verbose,
				test.includeWorkspaceSize,
				test.includeProgress,
				test.includeDuration,
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
			statusFilters, searchFilter, err := parseFilters(
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
			got := buildSeries(test.col, test.humanReadable)
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
