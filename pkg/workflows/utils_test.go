/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"reanahub/reana-client-go/pkg/formatter"
	"testing"
)

func TestGetWorkflowNameAndRunNumber(t *testing.T) {
	tests := map[string]struct {
		arg          string
		workflowName string
		runNumber    string
	}{
		"only name":            {arg: "foo", workflowName: "foo", runNumber: ""},
		"name and run number":  {arg: "foo.bar", workflowName: "foo", runNumber: "bar"},
		"run number with dots": {arg: "foo.bar.baz", workflowName: "foo", runNumber: "bar.baz"},
		"empty string":         {arg: "", workflowName: "", runNumber: ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			workflowName, runNumber := GetNameAndRunNumber(test.arg)
			if workflowName != test.workflowName {
				t.Errorf("Expected %s, got %s", test.workflowName, workflowName)
			}
			if runNumber != test.runNumber {
				t.Errorf("Expected %s, got %s", test.runNumber, runNumber)
			}
		})
	}
}

func TestGetWorkflowDuration(t *testing.T) {
	curTime := "2020-01-01T03:16:45"
	future := "2020-01-01T03:16:46"
	past := "2020-01-01T03:16:44"
	badFormat := "not_a_date"

	tests := map[string]struct {
		runStartedAt  *string
		runFinishedAt *string
		want          any
		wantError     bool
	}{
		"finished instantly":    {runStartedAt: &curTime, runFinishedAt: &curTime, want: 0.0},
		"finished in 1 second":  {runStartedAt: &curTime, runFinishedAt: &future, want: 1.0},
		"finished before start": {runStartedAt: &curTime, runFinishedAt: &past, want: -1.0},
		"nil arguments":         {runStartedAt: nil, runFinishedAt: nil, want: nil},
		"nil start":             {runStartedAt: nil, runFinishedAt: &curTime, want: nil},
		"nil finish":            {runStartedAt: &curTime, runFinishedAt: nil},
		"bad start format":      {runStartedAt: &badFormat, wantError: true},
		"bad finish format": {
			runStartedAt:  &curTime,
			runFinishedAt: &badFormat,
			wantError:     true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetDuration(test.runStartedAt, test.runFinishedAt)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if test.runStartedAt != nil && test.runFinishedAt == nil {
					duration, ok := got.(float64)
					if !ok || duration <= 0 {
						t.Errorf("Expected positive duration, got %v", got)
					}
				} else if got != test.want {
					t.Errorf("Expected %v, got %v", test.want, got)
				}
			}
		})
	}
}

func TestFormatSessionURI(t *testing.T) {
	tests := map[string]struct {
		serverURL string
		path      string
		token     string
		want      string
	}{
		"regular uri": {
			serverURL: "https://server.com",
			path:      "/api/",
			token:     "token",
			want:      "https://server.com/api/?token=token",
		},
		"no path": {
			serverURL: "https://server.com/",
			path:      "",
			token:     "token",
			want:      "https://server.com/?token=token",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatter.FormatSessionURI(test.serverURL, test.path, test.token)
			if got != test.want {
				t.Errorf("Expected %s, got %s", test.want, got)
			}
		})
	}
}
