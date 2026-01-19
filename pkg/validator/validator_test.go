/*
This file is part of REANA.
Copyright (C) 2022, 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package validator

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/spf13/pflag"
)

func TestValidateAccessToken(t *testing.T) {
	testNonEmptyString(t, ValidateAccessToken, InvalidAccessTokenMsg)
}

func TestValidateServerURL(t *testing.T) {
	testNonEmptyString(t, ValidateServerURL, InvalidServerURLMsg)
}

func TestValidateWorkflow(t *testing.T) {
	testNonEmptyString(t, ValidateWorkflow, InvalidWorkflowMsg)
}

func TestValidateChoice(t *testing.T) {
	choices := []string{"test1", "test2", "test3"}

	t.Run("invalid choice", func(t *testing.T) {
		invalidRes := ValidateChoice("invalid", choices, "test")
		expectedErr := "invalid value for 'test': 'invalid' is not part of 'test1', 'test2', 'test3'"
		if invalidRes == nil || invalidRes.Error() != expectedErr {
			t.Errorf("Expected: \"%s\", got: \"%v\"", expectedErr, invalidRes)
		}
	})

	t.Run("valid choice", func(t *testing.T) {
		validRes := ValidateChoice("test2", choices, "test")
		if validRes != nil {
			t.Errorf("Expected: \"%v\", got: \"%#v\"", nil, validRes)
		}
	})
}

func TestValidateAtLeastOne(t *testing.T) {
	tests := map[string]struct {
		flags     []pflag.Flag
		options   []string
		wantError bool
		expected  string
	}{
		"one matching": {
			flags:     []pflag.Flag{{Name: "option1", Changed: true}},
			options:   []string{"option1", "option2"},
			wantError: false,
		},
		"multiple matching": {
			flags: []pflag.Flag{
				{Name: "option1", Changed: true},
				{Name: "option2", Changed: true},
			},
			options:   []string{"option1", "option2"},
			wantError: false,
		},
		"matching not changed": {
			flags:     []pflag.Flag{{Name: "option1"}},
			options:   []string{"option1", "option2"},
			wantError: true,
			expected:  "at least one of the options: 'option1', 'option2' is required",
		},
		"empty flagset": {
			options:   []string{"option1", "option2"},
			wantError: true,
			expected:  "at least one of the options: 'option1', 'option2' is required",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			f := pflag.FlagSet{}
			for _, flag := range test.flags {
				f.AddFlag(&flag)
			}

			got := ValidateAtLeastOne(&f, test.options)
			if test.wantError {
				if got == nil {
					t.Errorf("Expected error: %s, got nil", test.expected)
				} else if got.Error() != test.expected {
					t.Errorf("Expected error: %s, got %s", test.expected, got.Error())
				}
			}
			if !test.wantError && got != nil {
				t.Errorf("Unexpected error: %s", got.Error())
			}
		})
	}
}

func TestValidateInputParameters(t *testing.T) {
	tests := map[string]struct {
		inputParams    map[string]string
		originalParams map[string]any
		expected       map[string]string
		expectedErrors []string
	}{
		"empty params": {
			inputParams: map[string]string{}, originalParams: map[string]any{},
			expected: map[string]string{},
		},
		"valid input": {
			inputParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
			originalParams: map[string]any{
				"param1": "value1",
				"param2": "value2",
			},
			expected: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		},
		"different original values": {
			inputParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
			originalParams: map[string]any{
				"param1": 1,
				"param2": false,
				"param3": "value",
			},
			expected: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		},
		"with invalid params": {
			inputParams: map[string]string{
				"param1":  "value1",
				"invalid": "value2",
			},
			originalParams: map[string]any{
				"param1": "value1",
				"param2": "value2",
			},
			expected: map[string]string{"param1": "value1"},
			expectedErrors: []string{
				"given parameter - invalid, is not in reana.yaml",
			},
		},
		"only invalid params": {
			inputParams: map[string]string{
				"invalid1": "value1",
				"invalid2": "value2",
			},
			originalParams: map[string]any{
				"param1": "value1",
				"param2": "value2",
			},
			expected: map[string]string{},
			expectedErrors: []string{
				"given parameter - invalid1, is not in reana.yaml",
				"given parameter - invalid2, is not in reana.yaml",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, errors := ValidateInputParameters(
				test.inputParams,
				test.originalParams,
			)
			if !reflect.DeepEqual(got, test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, got)
			}
			if len(test.expectedErrors) != len(errors) {
				t.Fatalf(
					"Expected errors: %v, got %v",
					test.expectedErrors,
					errors,
				)
			}
			for _, err := range errors {
				if !slices.Contains(test.expectedErrors, err.Error()) {
					t.Errorf(
						"Expected errors: %v, got unexpected '%s' error",
						test.expectedErrors,
						err.Error(),
					)
				}
			}
		})
	}
}

func TestValidateOperationalOptions(t *testing.T) {
	tests := map[string]struct {
		workflowType  string
		options       map[string]string
		expected      map[string]string
		wantError     bool
		expectedError string
	}{
		"empty options": {
			workflowType: "serial",
			options:      map[string]string{},
			expected:     map[string]string{},
		},
		"valid options": {
			workflowType: "serial",
			options:      map[string]string{"CACHE": "value", "FROM": "from"},
			expected:     map[string]string{"CACHE": "value", "FROM": "from"},
		},
		"valid with translation": {
			workflowType: "cwl",
			options:      map[string]string{"TARGET": "target"},
			expected:     map[string]string{"--target": "target"},
		},
		"invalid option": {
			workflowType:  "serial",
			options:       map[string]string{"INVALID": "value"},
			wantError:     true,
			expectedError: "operational option 'INVALID' not supported",
		},
		"invalid for workflow type": {
			workflowType: "serial",
			options: map[string]string{
				"CACHE":    "value",
				"toplevel": "level",
			},
			wantError:     true,
			expectedError: "operational option 'toplevel' not supported for serial workflows",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ValidateOperationalOptions(
				test.workflowType,
				test.options,
			)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != test.expectedError {
					t.Errorf("Expected error: '%s', got: '%s'", test.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: '%s'", err.Error())
				} else if !reflect.DeepEqual(got, test.expected) {
					t.Errorf("Expected %v, got %v", test.expected, got)
				}
			}
		})
	}
}

func TestValidateFile(t *testing.T) {
	tempDir := t.TempDir()
	emptyFile := tempDir + "/empty.txt"
	notReadableFile := tempDir + "/noperms.txt"
	_, err := os.Create(emptyFile)
	if err != nil {
		t.Fatalf("Error while creating empty file: %s", err.Error())
	}
	err = os.WriteFile(notReadableFile, []byte{}, 0222)
	if err != nil {
		t.Fatalf("Error while creating noperms file: %s", err.Error())
	}

	tests := map[string]struct {
		path      string
		wantError bool
		expected  string
	}{
		"existing file": {
			path: emptyFile,
		},
		"unexisting file": {
			path:      "this_doesnt_exist.txt",
			wantError: true,
			expected:  "file 'this_doesnt_exist.txt' does not exist",
		},
		"directory": {
			path:      tempDir,
			wantError: true,
			expected:  fmt.Sprintf("file '%s' is a directory", tempDir),
		},
		"not readable": {
			path:      notReadableFile,
			wantError: true,
			expected: fmt.Sprintf(
				"file '%s' is not readable",
				notReadableFile,
			),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := ValidateFile(test.path)
			if test.wantError {
				if got == nil {
					t.Errorf("Expected error: %s, got nil", test.expected)
				} else if got.Error() != test.expected {
					t.Errorf("Expected error: %s, got %s", test.expected, got.Error())
				}
			}
			if !test.wantError && got != nil {
				t.Errorf("Unexpected error: %s", got.Error())
			}
		})
	}
}

func testNonEmptyString(t *testing.T, f func(string) error, errorMsg string) {
	tests := map[string]struct {
		arg       string
		wantError bool
	}{
		"empty":        {arg: "", wantError: true},
		"white spaces": {arg: "   ", wantError: true},
		"valid":        {arg: "valid", wantError: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := f(test.arg)
			if test.wantError {
				if err == nil || err.Error() != errorMsg {
					t.Errorf("Expected: '%s', got: '%s'", errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected: 'nil', got: '%s'", err.Error())
				}
			}
		})
	}
}
