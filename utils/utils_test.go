/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/viper"

	"golang.org/x/exp/slices"
)

func TestGetRunStatuses(t *testing.T) {
	tests := []struct {
		includeDeleted bool
		numStatuses    int
	}{
		{includeDeleted: false, numStatuses: 7},
		{includeDeleted: true, numStatuses: 8},
	}
	for _, test := range tests {
		runStatuses := GetRunStatuses(test.includeDeleted)
		if len(runStatuses) != test.numStatuses {
			t.Errorf("Expected %d statuses, got %d", test.numStatuses, len(runStatuses))
		}

		if test.includeDeleted {
			if !slices.Contains(runStatuses, "deleted") {
				t.Errorf("Expected runStatuses to contain deleted")
			}
		} else if slices.Contains(runStatuses, "deleted") {
			t.Errorf("Expected runStatuses not to contain deleted")
		}
	}
}

func TestHasAnyPrefix(t *testing.T) {
	tests := []struct {
		prefixes []string
		str      string
		want     bool
	}{
		{prefixes: []string{"foo"}, str: "foobar", want: true},
		{prefixes: []string{"foo"}, str: "bar", want: false},
		{prefixes: []string{"foo", "bar"}, str: "foo", want: true},
		{prefixes: []string{"foo", "bar"}, str: "foobar", want: true},
		{prefixes: []string{"foo", "bar"}, str: "baz", want: false},
		{prefixes: []string{}, str: "foobar", want: false},
		{prefixes: []string{"foo", "bar"}, str: "", want: false},
	}
	for _, test := range tests {
		got := HasAnyPrefix(test.str, test.prefixes)
		if got != test.want {
			t.Errorf("Expected %t, got %t", test.want, got)
		}
	}
}

func TestFromIsoToTimestamp(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		dateIso   string
		wantError bool
		want      time.Time
	}{
		{
			dateIso: "2020-01-01T03:16:45",
			want:    time.Date(2020, 1, 1, 3, 16, 45, 0, time.UTC),
		},
		{
			dateIso: now.Format("2006-01-02T15:04:05"),
			want:    now.Truncate(time.Second),
		},
		{
			dateIso:   "2020-01-01T00:00:00Z",
			wantError: true,
		},
		{
			dateIso:   "09:30:00Z",
			wantError: true,
		},
		{
			dateIso:   "",
			wantError: true,
		},
	}
	for _, test := range tests {
		got, err := FromIsoToTimestamp(test.dateIso)
		if test.wantError {
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		} else {
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if got != test.want {
				t.Errorf("Expected %v, got %v", test.want, got)
			}
		}
	}
}

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
		workflowName, runNumber := GetWorkflowNameAndRunNumber(test.arg)
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
		got, err := GetWorkflowDuration(test.runStartedAt, test.runFinishedAt)
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
		got := FormatSessionURI(test.serverURL, test.path, test.token)
		if got != test.want {
			t.Errorf("Expected %s, got %s", test.want, got)
		}
	}
}

type testApiError struct {
	Payload struct{ Message string }
}

func (e *testApiError) Error() string { return e.Payload.Message }

func TestHandleApiError(t *testing.T) {
	serverURL := "https://localhost:8080"
	viper.Set("server-url", serverURL)
	t.Cleanup(func() {
		viper.Reset()
	})

	urlError := url.Error{}
	apiError := testApiError{Payload: struct{ Message string }{Message: "API Error"}}
	otherError := errors.New("other Error")

	tests := []struct {
		arg  error
		want string
	}{
		{
			arg: &urlError,
			want: fmt.Sprintf(
				"'%s' not found, please verify the provided server URL or check your internet connection",
				serverURL,
			),
		},
		{
			arg:  &apiError,
			want: apiError.Error(),
		},
		{
			arg:  otherError,
			want: otherError.Error(),
		},
	}
	for _, test := range tests {
		got := HandleApiError(test.arg)
		if got.Error() != test.want {
			t.Errorf("Expected %s, got %s", test.want, got)
		}
	}
}

func TestSplitLinesNoEmpty(t *testing.T) {
	tests := []struct {
		arg  string
		want []string
	}{
		{arg: "", want: []string{}},
		{arg: "a", want: []string{"a"}},
		{arg: "a\nb", want: []string{"a", "b"}},
		{arg: "a\nb\nc", want: []string{"a", "b", "c"}},
		{arg: "a\nb\nc\n", want: []string{"a", "b", "c"}},
		{arg: "a\n\nb\n\nc", want: []string{"a", "b", "c"}},
	}
	for _, test := range tests {
		got := SplitLinesNoEmpty(test.arg)
		if !slices.Equal(got, test.want) {
			t.Errorf("Expected %v, got %v", test.want, got)
		}
	}
}
