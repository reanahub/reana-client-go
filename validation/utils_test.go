/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package validation

import (
	"testing"
)

func TestValidateAccessToken(t *testing.T) {
	testNonEmptyString(t, ValidateAccessToken, invalidAccessTokenMsg)
}

func TestValidateServerURL(t *testing.T) {
	testNonEmptyString(t, ValidateServerURL, invalidServerURLMsg)
}

func TestValidateWorkflow(t *testing.T) {
	testNonEmptyString(t, ValidateWorkflow, invalidWorkflowMsg)
}

func TestValidateChoice(t *testing.T) {
	choices := []string{"test1", "test2", "test3"}

	invalidRes := ValidateChoice("invalid", choices, "test")
	expectedErr := "invalid value for 'test': 'invalid' is not part of 'test1', 'test2', 'test3'"
	if invalidRes == nil || invalidRes.Error() != expectedErr {
		t.Errorf("Expected: \"%s\", got: \"%v\"", expectedErr, invalidRes)
	}

	validRes := ValidateChoice("test2", choices, "test")
	if validRes != nil {
		t.Errorf("Expected: \"%v\", got: \"%#v\"", nil, validRes)
	}
}

func testNonEmptyString(t *testing.T, f func(string) error, errorMsg string) {
	for _, arg := range []string{"", "   "} {
		invalidRes := f(arg)
		if invalidRes == nil || invalidRes.Error() != errorMsg {
			t.Errorf("Expected: \"%s\", got: \"%v\"", errorMsg, invalidRes)
		}
	}

	validRes := f("valid")
	if validRes != nil {
		t.Errorf("Expected: \"%v\", got: \"%#v\"", nil, validRes)
	}
}
