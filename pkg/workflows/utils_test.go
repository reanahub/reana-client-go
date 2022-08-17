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
	tests := []struct {
		arg          string
		workflowName string
		runNumber    string
	}{
		{arg: "foo", workflowName: "foo", runNumber: ""},
		{arg: "foo.bar", workflowName: "foo", runNumber: "bar"},
		{arg: "foo.bar.baz", workflowName: "foo", runNumber: "bar.baz"},
		{arg: "", workflowName: "", runNumber: ""},
	}
	for _, test := range tests {
		workflowName, runNumber := GetNameAndRunNumber(test.arg)
		if workflowName != test.workflowName {
			t.Errorf("Expected %s, got %s", test.workflowName, workflowName)
		}
		if runNumber != test.runNumber {
			t.Errorf("Expected %s, got %s", test.runNumber, runNumber)
		}
	}
}

func TestGetWorkflowDuration(t *testing.T) {
	curTime := "2020-01-01T03:16:45"
	future := "2020-01-01T03:16:46"
	past := "2020-01-01T03:16:44"
	badFormat := "not_a_date"

	tests := []struct {
		runStartedAt  *string
		runFinishedAt *string
		want          any
		wantError     bool
	}{
		{runStartedAt: &curTime, runFinishedAt: &curTime, want: 0.0},
		{runStartedAt: &curTime, runFinishedAt: &future, want: 1.0},
		{runStartedAt: &curTime, runFinishedAt: &past, want: -1.0},
		{runStartedAt: nil, runFinishedAt: nil, want: nil},
		{runStartedAt: nil, runFinishedAt: &curTime, want: nil},
		{runStartedAt: &curTime, runFinishedAt: nil},
		{runStartedAt: &badFormat, wantError: true},
		{runStartedAt: &curTime, runFinishedAt: &badFormat, wantError: true},
	}
	for _, test := range tests {
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
	}
}

func TestFormatSessionURI(t *testing.T) {
	tests := []struct {
		serverURL string
		path      string
		token     string
		want      string
	}{
		{
			serverURL: "https://server.com",
			path:      "/api/",
			token:     "token",
			want:      "https://server.com/api/?token=token",
		},
		{
			serverURL: "https://server.com/",
			path:      "",
			token:     "token",
			want:      "https://server.com/?token=token",
		},
	}
	for _, test := range tests {
		got := formatter.FormatSessionURI(test.serverURL, test.path, test.token)
		if got != test.want {
			t.Errorf("Expected %s, got %s", test.want, got)
		}
	}
}
