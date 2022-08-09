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
	"os"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

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

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		hasToken           bool
		token              string
		hasServerURL       bool
		serverURL          string
		hasWorkflow        bool
		isWorkflowOptional bool
		workflow           string
		wantError          bool
		errorMsg           string
	}{
		{
			hasToken: true, token: "",
			hasServerURL: false, hasWorkflow: false,
			wantError: true, errorMsg: validation.InvalidAccessTokenMsg,
		},
		{
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "",
			hasWorkflow: false,
			wantError:   true, errorMsg: validation.InvalidServerURLMsg,
		},
		{
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "https://localhost:8080",
			hasWorkflow: false, wantError: false,
		},
		{
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: false, workflow: "",
			wantError: true, errorMsg: validation.InvalidWorkflowMsg,
		},
		{
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: true,
			workflow: "", wantError: false,
		},
		{
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: false,
			workflow: "workflow", wantError: false,
		},
		{
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "https://localhost:8080",
			hasWorkflow: true, isWorkflowOptional: false,
			workflow: "workflow", wantError: false,
		},
	}

	for _, test := range tests {
		cmd := NewRootCmd()
		f := cmd.Flags()
		if test.hasToken {
			f.String("access-token", test.token, "")
		}
		if test.hasServerURL {
			viper.Set("server-url", test.serverURL)
		}
		if test.hasWorkflow {
			f.String("workflow", test.workflow, "")
			if test.isWorkflowOptional {
				err := f.SetAnnotation("workflow", "properties", []string{"optional"})
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		err := validateFlags(cmd)
		if test.wantError {
			if err == nil {
				t.Error("Expected error, instead got nil")
			} else if err.Error() != test.errorMsg {
				t.Errorf("Expected '%s' in error output, instead got '%s'", test.errorMsg, err.Error())
			}
		} else if err != nil {
			t.Errorf("Got unexpected error '%s'", err.Error())
		}

		viper.Reset()
	}
}

func TestSetupViper(t *testing.T) {
	tests := []struct {
		env       string
		viperProp string
		value     string
	}{
		{env: "REANA_SERVER_URL", viperProp: "server-url", value: "https://localhost:8080"},
		{env: "REANA_ACCESS_TOKEN", viperProp: "access-token", value: "1234"},
		{env: "REANA_WORKON", viperProp: "workflow", value: "workflow"},
	}

	for _, test := range tests {
		err := os.Setenv(test.env, test.value)
		if err != nil {
			t.Fatal(err)
		}
		err = setupViper(nil)
		if err != nil {
			t.Fatal(err)
		}

		viperValue := viper.GetString(test.viperProp)
		if viperValue != test.value {
			t.Errorf(
				"Expected '%s' to be '%s', instead got '%s'",
				test.viperProp,
				test.value,
				viperValue,
			)
		}

		viper.Reset()
	}
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		level   string
		isValid bool
	}{
		{level: "DEBUG", isValid: true},
		{level: "INFO", isValid: true},
		{level: "QUIET", isValid: false},
	}

	for _, test := range tests {
		err := setupLogger(test.level)
		if test.isValid {
			if err != nil {
				t.Errorf("Got unexpected error '%s'", err.Error())
			} else {
				loglevel := log.GetLevel().String()
				if loglevel != strings.ToLower(test.level) {
					t.Errorf("Expected log level '%s', instead got '%s'", test.level, loglevel)
				}
			}
		} else if err == nil {
			t.Error("Expected error, instead got nil")
		}
	}
}
