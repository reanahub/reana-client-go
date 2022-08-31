/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package internal provides helper functions for testing.
package internal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"reanahub/reana-client-go/pkg/errorhandler"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ExecuteCommand executes a cobra command with the given args.
// Returns the output of the command and any error it may provide.
func ExecuteCommand(cmd *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.String(), errorhandler.HandleApiError(err)
}

// TestCmdParams stores information to be used when testing command outputs.
type TestCmdParams struct {
	Cmd             *cobra.Command
	ServerResponses map[string]ServerResponse
	Args            []string
	Expected        []string
	Unwanted        []string
	WantError       bool
}

// ServerResponse represents the response of the server in an API call.
type ServerResponse struct {
	StatusCode   int
	ResponseFile string
}

// TestCmdRun tests a command with the parameters given by p. Check TestCmdParams to know the possible options.
func TestCmdRun(t *testing.T, p TestCmdParams) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accessToken := r.URL.Query().Get("access_token"); accessToken != "1234" {
			t.Errorf("Expected access token '1234', got '%v'", accessToken)
		}
		res, validPath := p.ServerResponses[r.URL.Path]
		if validPath {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(res.StatusCode)

			var body []byte
			if res.ResponseFile != "" {
				var err error
				body, err = os.ReadFile(res.ResponseFile)
				if err != nil {
					t.Fatalf("Error while reading response file: %v", err)
				}
			}
			_, err := w.Write(body)
			if err != nil {
				t.Fatalf("Error while writing response body: %v", err)
			}
		} else {
			t.Fatalf("Unexpected request to '%v'", r.URL.Path)
		}
	}))

	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		server.Close()
		viper.Reset()
	})

	args := append([]string{"-t", "1234"}, p.Args...)
	output, err := ExecuteCommand(p.Cmd, args...)

	if !p.WantError && err != nil {
		t.Fatalf("Got unexpected error '%s'", err.Error())
	}
	if p.WantError && err == nil {
		t.Fatalf("Expected error, instead got '%s'", output)
	}

	for _, test := range p.Expected {
		if !p.WantError && !strings.Contains(output, test) {
			t.Errorf("Expected '%s' in output, instead got '%s'", test, output)
		}
		if p.WantError && !strings.Contains(err.Error(), test) && !strings.Contains(output, test) {
			t.Errorf("Expected '%s' in error output, instead got '%s'", test, err.Error())
		}
	}

	for _, forbidden := range p.Unwanted {
		if !p.WantError && strings.Contains(output, forbidden) {
			t.Errorf("Expected '%s' not to be in output, instead got '%s'", forbidden, output)
		}
		if p.WantError && (strings.Contains(err.Error(), forbidden) ||
			strings.Contains(output, forbidden)) {
			t.Errorf(
				"Expected '%s' not to be in error output, instead got '%s'",
				forbidden,
				err.Error(),
			)
		}
	}
}
