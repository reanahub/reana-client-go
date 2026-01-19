/*
This file is part of REANA.
Copyright (C) 2022, 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package errorhandler

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/spf13/viper"
)

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
	apiError := testApiError{
		Payload: struct{ Message string }{Message: "API Error"},
	}
	otherError := errors.New("other Error")

	tests := map[string]struct {
		arg  error
		want string
	}{
		"server not found": {
			arg: &urlError,
			want: fmt.Sprintf(
				"'%s' not found, please verify the provided server URL or check your internet connection",
				serverURL,
			),
		},
		"api error": {
			arg:  &apiError,
			want: apiError.Error(),
		},
		"other error": {
			arg:  otherError,
			want: otherError.Error(),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := HandleApiError(test.arg)
			if got.Error() != test.want {
				t.Errorf("Expected %s, got %s", test.want, got)
			}
		})
	}
}
