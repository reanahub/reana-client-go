/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"reanahub/reana-client-go/utils"
	"testing"

	"github.com/go-gota/gota/series"

	"golang.org/x/exp/slices"
)

var listServerPath = "/api/workflows"

func TestList(t *testing.T) {
	successResponse := `{
		"total": 2,
		"items": [
			{
				"created": "2022-07-28T12:04:37",
				"id": "my_workflow_id",
				"launcher_url": "https://test.test/url",
				"name": "my_workflow.23",
				"progress": {
					"finished": {
						"job_ids": ["job1", "job2"],
						"total": 2
					},
					"total": {
						"job_ids": [],
						"total": 2
					},
					"run_finished_at": "2022-07-28T12:13:10",
					"run_started_at": "2022-07-28T12:04:52"
				},
				"size": {
					"human_readable": "1 KiB",
					"raw": 1024
				},
				"status": "finished",
				"user": "user",
				"session_status": "created",
				"session_type": "jupyter",
				"session_uri": "/session1uri"
			},
			{
				"created": "2022-08-10T17:14:12",
				"id": "my_workflow2_id",
				"launcher_url": "https://test.test/url2",
				"name": "my_workflow2.12",
				"progress": {
					"finished": {
						"job_ids": ["job3"],
						"total": 1
					},
					"total": {
						"job_ids": [],
						"total": 2
					},
					"run_finished_at": null,
					"run_started_at": "2022-08-10T18:04:52"
				},
				"size": {
					"human_readable": "",
					"raw": -1
				},
				"status": "running",
				"user": "user",
				"session_status": "created",
				"session_type": "jupyter",
				"session_uri": "/session2uri"
			}
		]
	}`

	tests := map[string]TestCmdParams{
		"default": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
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
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-s"},
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
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--format", "name,status"},
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
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--format", "name=my_workflow,status"},
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
		"json": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--json"},
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
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-v"},
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
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--include-workspace-size"},
			expected:       []string{"SIZE", "1024", " -1 "},
			unwanted:       []string{"1 KiB"},
		},
		"human readable size": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--include-workspace-size", "-r"},
			expected:       []string{"SIZE", "1 KiB"},
			unwanted:       []string{"1024", " -1 "},
		},
		"include duration": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--include-duration"},
			expected:       []string{"DURATION", "498"},
			unwanted: []string{
				"ID", "USER", "SIZE", "PROGRESS", "SESSION_TYPE",
				"SESSION_URI", "SESSION_STATUS",
			},
		},
		"include progress": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--include-progress"},
			expected:       []string{"PROGRESS", "2/2", "1/2"},
			unwanted: []string{
				"ID", "USER", "SIZE", "DURATION", "SESSION_TYPE",
				"SESSION_URI", "SESSION_STATUS",
			},
		},
		"sorted": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--sort", "run_number"},
			expected:       []string{"STATUS   \n my_workflow "},
		},
		"malformed filters": {
			args: []string{"--filter", "name"},
			expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			wantError: true,
		},
		"unexisting workflow": {
			serverResponse: `{"message": "REANA_WORKON is set to invalid, but that workflow does not exist."}`,
			statusCode:     http.StatusNotFound,
			args:           []string{"-w", "invalid"},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
		"invalid size": {
			serverResponse: `{"message": "Field 'size': Must be at least 1."}`,
			statusCode:     http.StatusBadRequest,
			args:           []string{"--size", "0"},
			expected:       []string{"Field 'size': Must be at least 1."},
			wantError:      true,
		},
		"invalid sort columns": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--sort", "invalid"},
			expected: []string{
				"Warning: sort operation was aborted, column 'invalid' does not exist",
				"NAME", "RUN_NUMBER", "CREATED", "STARTED", "ENDED", "STATUS",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "list"
			params.serverPath = listServerPath
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
			header := buildListHeader(
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
			statusFilters: utils.GetRunStatuses(false),
			searchFilter:  "",
		},
		"with deleted runs": {
			filterInput:     []string{},
			showDeletedRuns: true,
			statusFilters:   utils.GetRunStatuses(true),
			searchFilter:    "",
		},
		"with show all": {
			filterInput:   []string{},
			showAll:       true,
			statusFilters: utils.GetRunStatuses(true),
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
