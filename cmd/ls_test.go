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
	"testing"

	"github.com/go-gota/gota/series"
)

var lsPathTemplate = "/api/workflows/%s/workspace"

func TestLs(t *testing.T) {
	workflowName := "my_workflow"
	successResponse := `{
		"items": [
			{
				"last-modified": "2022-07-11T12:50:33",
				"name": "code/gendata.C",
				"size": {
					"human_readable": "1.89 KiB",
					"raw": 1937
				}
			},
			{
				"last-modified": "2022-07-11T13:30:17",
				"name": "results/data.root",
				"size": {
					"human_readable": "150.83 KiB",
					"raw": 154455
				}
			}
		],
		"total": 4
}`
	tests := map[string]TestCmdParams{
		"default": {
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName},
			expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1937", "2022-07-11T12:50:33",
				"results/data.root", "154455", "2022-07-11T13:30:17",
			},
		},
		"human readable": {
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "-r"},
			expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1.89 KiB", "2022-07-11T12:50:33",
				"results/data.root", "150.83 KiB", "2022-07-11T13:30:17",
			},
		},
		"files in black list": {
			serverPath: fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: `{
				"items": [
					{
						"last-modified": "2022-07-11T12:50:33",
						"name": ".git/test.C",
						"size": {
							"human_readable": "1.89 KiB",
							"raw": 1937
						}
					},
					{
						"last-modified": "2022-07-11T13:30:17",
						"name": "results/data.root",
						"size": {
							"human_readable": "150.83 KiB",
							"raw": 154455
						}
					}
				],
				"total": 4
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName},
			expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"results/data.root", "154455", "2022-07-11T13:30:17",
			},
			unwanted: []string{
				".git/test.C", "1937", "2022-07-11T12:50:33",
			},
		},
		"format columns": {
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--format", "name,last-modified"},
			expected: []string{
				"NAME", "LAST-MODIFIED",
				"code/gendata.C", "2022-07-11T12:50:33",
				"results/data.root", "2022-07-11T13:30:17",
			},
			unwanted: []string{
				"SIZE", "1937", "154455",
			},
		},
		"format with filters": {
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--format", "name=code/gendata.C"},
			expected: []string{
				"NAME", "code/gendata.C",
			},
			unwanted: []string{
				"SIZE", "LAST-MODIFIED",
				"1937", "2022-07-11T12:50:33",
				"results/data.root", "154455", "2022-07-11T13:30:17",
			},
		},
		"json": {
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--json"},
			expected: []string{`[
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
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "--url"},
			expected: []string{
				fmt.Sprintf("/api/workflows/%s/workspace/code/gendata.C", workflowName),
				fmt.Sprintf("/api/workflows/%s/workspace/results/data.root", workflowName),
			},
		},
		"with filters": {
			serverPath: fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: `{
				"items": [
					{
						"last-modified": "2022-07-11T12:50:33",
						"name": "code/gendata.C",
						"size": {
							"human_readable": "1.89 KiB",
							"raw": 1937
						}
					}
				],
				"total": 4
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName, "--filter", "name=code/gendata.C"},
			expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1937", "2022-07-11T12:50:33",
			},
		},
		"filename arg": {
			serverPath: fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: `{
				"items": [
					{
						"last-modified": "2022-07-11T12:50:33",
						"name": "code/gendata.C",
						"size": {
							"human_readable": "1.89 KiB",
							"raw": 1937
						}
					}
				],
				"total": 4
			}`,
			statusCode: http.StatusOK,
			args:       []string{"-w", workflowName, "code/gendata.C"},
			expected: []string{
				"NAME", "SIZE", "LAST-MODIFIED",
				"code/gendata.C", "1937", "2022-07-11T12:50:33",
			},
		},
		"malformed filters": {
			serverPath: fmt.Sprintf(lsPathTemplate, workflowName),
			args:       []string{"-w", workflowName, "--filter", "name"},
			expected: []string{
				"wrong input format. Please use --filter filter_name=filter_value",
			},
			wantError: true,
		},
		"unexisting workflow": {
			serverPath:     fmt.Sprintf(lsPathTemplate, "invalid"),
			serverResponse: `{"message": "REANA_WORKON is set to invalid, but that workflow does not exist."}`,
			statusCode:     http.StatusNotFound,
			args:           []string{"-w", "invalid"},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
		"invalid size": {
			serverPath:     fmt.Sprintf(lsPathTemplate, workflowName),
			serverResponse: `{"message": "Field 'size': Must be at least 1."}`,
			statusCode:     http.StatusBadRequest,
			args:           []string{"-w", workflowName, "--size", "0"},
			expected:       []string{"Field 'size': Must be at least 1."},
			wantError:      true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "ls"
			testCmdRun(t, params)
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
			got := buildLsSeries(test.col, test.humanReadable)
			if got.Name != test.want.Name {
				t.Errorf("series has name '%s', wanted '%s'", got.Name, test.want.Name)
			}
			if got.Type() != test.want.Type() {
				t.Errorf("series has type '%s', wanted '%s'", got.Type(), test.want.Type())
			}
		})
	}
}
