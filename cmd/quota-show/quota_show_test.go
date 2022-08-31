/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package quota_show

import (
	"bytes"
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

var quotaShowServerPath = "/api/you"

func TestQuotaShow(t *testing.T) {
	tests := map[string]TestCmdParams{
		"show resources": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resources"},
			Expected: []string{"cpu", "disk"},
			Unwanted: []string{"used", "limit", "usage"},
		},
		"cpu limit": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "cpu", "--report", "limit"},
			Expected: []string{"100"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk"},
		},
		"cpu usage": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "cpu", "--report", "usage"},
			Expected: []string{"10"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk", "100"},
		},
		"cpu limit human": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "cpu", "--report", "limit", "-h"},
			Expected: []string{"10m 50s"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk", "100"},
		},
		"cpu usage human": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "cpu", "--report", "usage", "-h"},
			Expected: []string{"1m 5s"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk", "10"},
		},
		"cpu all reports": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "cpu"},
			Expected: []string{"10 out of 100 used (10%)"},
			Unwanted: []string{"limit", "usage", "cpu", "disk"},
		},
		"cpu limit no info": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:     []string{"--resource", "cpu", "--report", "limit"},
			Expected: []string{"No limit"},
			Unwanted: []string{"used", "usage", "cpu", "disk"},
		},
		"cpu usage no info": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:     []string{"--resource", "cpu", "--report", "usage"},
			Expected: []string{"No usage"},
			Unwanted: []string{"used", "limit", "cpu", "disk"},
		},
		"cpu all reports no info": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:     []string{"--resource", "cpu"},
			Expected: []string{"0 used"},
			Unwanted: []string{"limit", "usage", "cpu", "disk", "out of", "%"},
		},
		"disk limit": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "disk", "--report", "limit"},
			Expected: []string{"200"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk"},
		},
		"disk usage": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "disk", "--report", "usage"},
			Expected: []string{"20"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk", "200"},
		},
		"disk limit human": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "disk", "--report", "limit", "-h"},
			Expected: []string{"20 MiB"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk", "200"},
		},
		"disk usage human": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "disk", "--report", "usage", "-h"},
			Expected: []string{"2 MiB"},
			Unwanted: []string{"used", "limit", "usage", "cpu", "disk", "20"},
		},
		"disk all reports": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args:     []string{"--resource", "disk"},
			Expected: []string{"20 out of 200 used (10%)"},
			Unwanted: []string{"limit", "usage", "cpu", "disk"},
		},
		"disk limit no info": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:     []string{"--resource", "disk", "--report", "limit"},
			Expected: []string{"No limit"},
			Unwanted: []string{"used", "usage", "cpu", "disk"},
		},
		"disk usage no info": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:     []string{"--resource", "disk", "--report", "usage"},
			Expected: []string{"No usage"},
			Unwanted: []string{"used", "limit", "cpu", "disk"},
		},
		"disk all reports no info": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:     []string{"--resource", "disk"},
			Expected: []string{"0 used"},
			Unwanted: []string{"limit", "usage", "cpu", "disk", "out of", "%"},
		},
		"no resources specified": {
			Args: []string{}, WantError: true,
			Expected: []string{
				"at least one of the options: 'resource', 'resources' is required",
				"Usage",
			},
		},
		"invalid report value": {
			Args: []string{"--resource", "cpu", "--report", "invalid"}, WantError: true,
			Expected: []string{fmt.Sprintf(
				"invalid value for 'report': 'invalid' is not part of '%s'",
				strings.Join(config.QuotaReports, "', '"),
			)},
		},
		"invalid resource": {
			ServerResponses: map[string]ServerResponse{
				quotaShowServerPath: {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/no_info.json",
				},
			},
			Args:      []string{"--resource", "invalid"},
			WantError: true,
			Expected: []string{
				"resource 'invalid' is not valid\nAvailable resources are",
				"cpu",
				"disk",
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
