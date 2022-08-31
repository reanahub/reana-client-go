/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package ls

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"

	"github.com/go-gota/gota/series"
)

var lsPathTemplate = "/api/workflows/%s/workspace"

func TestLs(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1937", "2022-07-11T12:50:33",
				"results/data.root", "154455", "2022-07-11T13:30:17",
			},
		},
		"human readable": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "-h"},
			Expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1.89 KiB", "2022-07-11T12:50:33",
				"results/data.root", "150.83 KiB", "2022-07-11T13:30:17",
			},
		},
		"files in black list": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/blacklist.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"results/data.root", "154455", "2022-07-11T13:30:17",
			},
			Unwanted: []string{
				".git/test.C", "1937", "2022-07-11T12:50:33",
			},
		},
		"format columns": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--format", "name,last-modified"},
			Expected: []string{
				"NAME", "LAST-MODIFIED",
				"code/gendata.C", "2022-07-11T12:50:33",
				"results/data.root", "2022-07-11T13:30:17",
			},
			Unwanted: []string{
				"SIZE", "1937", "154455",
			},
		},
		"format with filters": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--format", "name=code/gendata.C"},
			Expected: []string{
				"NAME", "code/gendata.C",
			},
			Unwanted: []string{
				"SIZE", "LAST-MODIFIED",
				"1937", "2022-07-11T12:50:33",
				"results/data.root", "154455", "2022-07-11T13:30:17",
			},
		},
		"invalid format column": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--format", "invalid"},
			Expected: []string{
				"invalid value for 'format column': 'invalid' is not part of 'name', 'size', 'last-modified'",
			},
			WantError: true,
		},
		"json": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--json"},
			Expected: []string{`[
  {
    "last-modified": "2022-07-11T12:50:33",
    "name": "code/gendata.C",
    "size": 1937
  },
  {
    "last-modified": "2022-07-11T13:30:17",
    "name": "results/data.root",
    "size": 154455
  }
]`},
		},
		"display URLs": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--url"},
			Expected: []string{
				fmt.Sprintf("/api/workflows/%s/workspace/code/gendata.C", workflowName),
				fmt.Sprintf("/api/workflows/%s/workspace/results/data.root", workflowName),
			},
		},
		"with filters": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/filters.json",
				},
			},
			Args: []string{"-w", workflowName, "--filter", "name=code/gendata.C"},
			Expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1937", "2022-07-11T12:50:33",
			},
		},
		"filename arg": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/filters.json",
				},
			},
			Args: []string{"-w", workflowName, "code/gendata.C"},
			Expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1937", "2022-07-11T12:50:33",
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
				fmt.Sprintf(lsPathTemplate, "invalid"): {
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
				fmt.Sprintf(lsPathTemplate, workflowName): {
					StatusCode:   http.StatusBadRequest,
					ResponseFile: "../../testdata/inputs/invalid_size.json",
				},
			},
			Args:      []string{"-w", workflowName, "--size", "0"},
			Expected:  []string{"Field 'size': Must be at least 1."},
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

func TestBuildLsSeries(t *testing.T) {
	tests := map[string]struct {
		col           string
		humanReadable bool
		want          series.Series
	}{
		"regular column": {
			col: "name", humanReadable: false, want: series.New([]string{}, series.String, "name"),
		},
		"raw size": {
			col: "size", humanReadable: false, want: series.New([]int{}, series.Int, "size"),
		},
		"human readable size": {
			col: "size", humanReadable: true, want: series.New([]string{}, series.String, "size"),
		},
		"human readable other column": {
			col: "name", humanReadable: true, want: series.New([]string{}, series.String, "name"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildSeries(test.col, test.humanReadable)
			if got.Name != test.want.Name {
				t.Errorf("series has name '%s', wanted '%s'", got.Name, test.want.Name)
			}
			if got.Type() != test.want.Type() {
				t.Errorf("series has type '%s', wanted '%s'", got.Type(), test.want.Type())
			}
		})
	}
}
