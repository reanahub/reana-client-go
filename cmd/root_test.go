/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"reanahub/reana-client-go/utils"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func testCmdRun(
	t *testing.T,
	cmd, serverPath, serverResponse string,
	wantError bool, expectedMsgs []string,
	args ...string,
) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accessToken := r.URL.Query().Get("access_token"); accessToken != "1234" {
			t.Errorf("Expected access token '1234', got '%v'", accessToken)
		}
		if r.URL.Path == serverPath {
			w.Header().Add("Content-Type", "application/json")
			_, err := w.Write([]byte(serverResponse))
			if err != nil {
				t.Fatalf("Error while writing response body: %v", err)
			}
		} else {
			t.Fatalf("Unexpected request to '%v'", r.URL.Path)
		}
	}))
	defer server.Close()

	viper.Set("server-url", server.URL)
	defer viper.Reset()

	rootCmd := NewRootCmd()
	args = append([]string{cmd, "-t", "1234"}, args...)
	output, err := utils.ExecuteCommand(rootCmd, args...)

	if !wantError && err != nil {
		t.Fatalf("Got unexpected error '%s'", err.Error())
	}
	if wantError && err == nil {
		t.Fatalf("Expected error, instead got '%s'", output)
	}

	for _, test := range expectedMsgs {
		if !wantError && !strings.Contains(output, test) {
			t.Errorf("Expected '%s' in output, instead got '%s'", test, output)
		}
		if wantError && !strings.Contains(err.Error(), test) {
			t.Errorf("Expected '%s' in error output, instead got '%s'", test, err.Error())
		}
	}
}
