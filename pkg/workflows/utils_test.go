/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"testing"
)

func TestGetNameAndRunNumber(t *testing.T) {
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

func TestGetDuration(t *testing.T) {
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

func TestGetLastCommand(t *testing.T) {
	emptyString := ""
	echoHello := "echo 'hello'"
	echoWorld := "echo 'hello'\n\n echo 'world'"
	step1 := "step1"
	bashCmd := "bash -c \"cd folder; ls \""

	tests := map[string]struct {
		lastCommand *string
		stepName    *string
		expected    string
	}{
		"both nil": {
			lastCommand: nil,
			stepName:    nil,
			expected:    "-",
		},
		"lastCommand empty": {
			lastCommand: &emptyString,
			stepName:    nil,
			expected:    "-",
		},
		"stepName empty": {
			lastCommand: nil,
			stepName:    &emptyString,
			expected:    "-",
		},
		"both empty": {
			lastCommand: &emptyString,
			stepName:    &emptyString,
			expected:    "-",
		},
		"valid lastCommand": {
			lastCommand: &echoHello,
			stepName:    nil,
			expected:    "echo 'hello'",
		},
		"valid stepName": {
			lastCommand: nil,
			stepName:    &step1,
			expected:    "step1",
		},
		"newlines in lastCommand": {
			lastCommand: &echoWorld,
			stepName:    nil,
			expected:    "echo 'hello';  echo 'world'",
		},
		"command with prefix": {
			lastCommand: &bashCmd,
			stepName:    &emptyString,
			expected:    "ls",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetLastCommand(test.lastCommand, test.stepName)
			if got != test.expected {
				t.Errorf("Test %s: Expected %q, got %q", name, test.expected, got)
			}
		})
	}
}

func TestStatusChangeMessage(t *testing.T) {
	tests := map[string]struct {
		workflow  string
		status    string
		expected  string
		wantError bool
	}{
		"running": {
			workflow: "workflow",
			status:   "running",
			expected: "workflow is running",
		},
		"pending": {
			workflow: "workflow",
			status:   "pending",
			expected: "workflow is pending",
		},
		"deleted": {
			workflow: "workflow",
			status:   "deleted",
			expected: "workflow has been deleted",
		},
		"created": {
			workflow: "workflow",
			status:   "created",
			expected: "workflow has been created",
		},
		"stopped": {
			workflow: "workflow",
			status:   "stopped",
			expected: "workflow has been stopped",
		},
		"queued": {
			workflow: "workflow",
			status:   "queued",
			expected: "workflow has been queued",
		},
		"finished": {
			workflow: "workflow",
			status:   "finished",
			expected: "workflow has finished",
		},
		"failed": {
			workflow: "workflow",
			status:   "failed",
			expected: "workflow has failed",
		},
		"invalid status": {
			workflow:  "workflow",
			status:    "invalid",
			expected:  "unrecognised status invalid",
			wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := StatusChangeMessage(test.workflow, test.status)
			if err == nil {
				if test.wantError {
					t.Errorf("Expected error, got nil")
				} else if got != test.expected {
					t.Errorf("Expected %s, got %s", test.expected, got)
				}
			} else {
				if !test.wantError {
					t.Errorf("Expected no error, got %s", err.Error())
				} else if err.Error() != test.expected {
					t.Errorf("Expected %s error, got %s", test.expected, err.Error())
				}
			}
		})
	}
}
