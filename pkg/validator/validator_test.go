/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package validator

import (
	"testing"
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
