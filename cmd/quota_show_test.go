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
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

var quotaShowServerPath = "/api/you"

func TestQuotaShow(t *testing.T) {
	successResponse := `{
		"quota": {
			"cpu": {
				"health": "healthy",
				"usage": {
					"human_readable": "1m 5s",
					"raw": 10
				},
				"limit": {
					"human_readable": "10m 50s",
					"raw": 100
				}
			},
			"disk": {
				"health": "healthy",
				"usage": {
					"human_readable": "2 MiB",
					"raw": 20
				},
				"limit": {
					"human_readable": "20 MiB",
					"raw": 200
				}
			}
		}
	}`
	noInfoResponse := `{
		"quota": {
			"cpu": {},
			"disk": {}
		}
	}`

	tests := map[string]TestCmdParams{
		"show resources": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resources"},
			expected:       []string{"cpu", "disk"},
			unwanted:       []string{"used", "limit", "usage"},
		},
		"cpu limit": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu", "--report", "limit"},
			expected:       []string{"100"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk"},
		},
		"cpu usage": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu", "--report", "usage"},
			expected:       []string{"10"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk", "100"},
		},
		"cpu limit human": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu", "--report", "limit", "-r"},
			expected:       []string{"10m 50s"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk", "100"},
		},
		"cpu usage human": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu", "--report", "usage", "-r"},
			expected:       []string{"1m 5s"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk", "10"},
		},
		"cpu all reports": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu"},
			expected:       []string{"10 out of 100 used (10%)"},
			unwanted:       []string{"limit", "usage", "cpu", "disk"},
		},
		"cpu limit no info": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu", "--report", "limit"},
			expected:       []string{"No limit"},
			unwanted:       []string{"used", "usage", "cpu", "disk"},
		},
		"cpu usage no info": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu", "--report", "usage"},
			expected:       []string{"No usage"},
			unwanted:       []string{"used", "limit", "cpu", "disk"},
		},
		"cpu all reports no info": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "cpu"},
			expected:       []string{"0 used"},
			unwanted:       []string{"limit", "usage", "cpu", "disk", "out of", "%"},
		},
		"disk limit": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk", "--report", "limit"},
			expected:       []string{"200"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk"},
		},
		"disk usage": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk", "--report", "usage"},
			expected:       []string{"20"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk", "200"},
		},
		"disk limit human": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk", "--report", "limit", "-r"},
			expected:       []string{"20 MiB"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk", "200"},
		},
		"disk usage human": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk", "--report", "usage", "-r"},
			expected:       []string{"2 MiB"},
			unwanted:       []string{"used", "limit", "usage", "cpu", "disk", "20"},
		},
		"disk all reports": {
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk"},
			expected:       []string{"20 out of 200 used (10%)"},
			unwanted:       []string{"limit", "usage", "cpu", "disk"},
		},
		"disk limit no info": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk", "--report", "limit"},
			expected:       []string{"No limit"},
			unwanted:       []string{"used", "usage", "cpu", "disk"},
		},
		"disk usage no info": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk", "--report", "usage"},
			expected:       []string{"No usage"},
			unwanted:       []string{"used", "limit", "cpu", "disk"},
		},
		"disk all reports no info": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "disk"},
			expected:       []string{"0 used"},
			unwanted:       []string{"limit", "usage", "cpu", "disk", "out of", "%"},
		},
		"no resources specified": {
			args: []string{}, wantError: true,
			expected: []string{fmt.Sprintf(
				"at least one of the options: '%s' is required",
				strings.Join([]string{"resource", "resources"}, "', '"),
			)},
		},
		"invalid report value": {
			args: []string{"--resource", "cpu", "--report", "invalid"}, wantError: true,
			expected: []string{fmt.Sprintf(
				"invalid value for 'report': 'invalid' is not part of '%s'",
				strings.Join(config.QuotaReports, "', '"),
			)},
		},
		"invalid resource": {
			serverResponse: noInfoResponse,
			statusCode:     http.StatusOK,
			args:           []string{"--resource", "invalid"},
			wantError:      true,
			expected: []string{
				"resource 'invalid' is not valid\nAvailable resources are",
				"cpu",
				"disk",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "quota-show"
			params.serverPath = quotaShowServerPath
			testCmdRun(t, params)
		})
	}
}

func TestDisplayQuotaResourceUsage(t *testing.T) {
	tests := map[string]struct {
		health        string
		usageHuman    string
		limitHuman    string
		usageRaw      float64
		limitRaw      float64
		humanReadable bool
		expected      string
		expectedColor text.Color
	}{
		"raw usage": {
			usageRaw: 123, limitRaw: 0,
			expected: "123 used", expectedColor: text.Reset,
		},
		"human readable usage": {
			usageHuman: "2 MiB", limitRaw: 0, humanReadable: true,
			expected: "2 MiB used", expectedColor: text.Reset,
		},
		"raw with limit": {
			usageRaw: 10, limitRaw: 100, health: "healthy",
			expected: "10 out of 100 used (10%)", expectedColor: displayer.ResourceHealthToColor["healthy"],
		},
		"human readable with limit": {
			usageRaw: 10, limitRaw: 100, health: "healthy",
			usageHuman: "1 MiB", limitHuman: "10 MiB", humanReadable: true,
			expected: "1 MiB out of 10 MiB used (10%)", expectedColor: displayer.ResourceHealthToColor["healthy"],
		},
		"warning health": {
			usageRaw: 70, limitRaw: 100, health: "warning",
			expected: "70 out of 100 used (70%)", expectedColor: displayer.ResourceHealthToColor["warning"],
		},
		"critical health": {
			usageRaw: 95, limitRaw: 100, health: "critical",
			expected: "95 out of 100 used (95%)", expectedColor: displayer.ResourceHealthToColor["critical"],
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			displayQuotaResourceUsage(
				test.health,
				quotaResourceStat{HumanReadable: test.usageHuman, Raw: test.usageRaw},
				quotaResourceStat{HumanReadable: test.limitHuman, Raw: test.limitRaw},
				test.humanReadable, buf,
			)

			got := buf.String()
			testBuf := new(bytes.Buffer)
			displayer.PrintColorable(test.expected+"\n", testBuf, test.expectedColor)
			expected := testBuf.String()
			if got != expected {
				t.Errorf("expected %s, got %s", expected, got)
			}
		})
	}
}
