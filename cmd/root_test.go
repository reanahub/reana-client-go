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

	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		server.Close()
		viper.Reset()
	})

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
	tests := map[string]struct {
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
		"invalid token": {
			hasToken: true, token: "",
			hasServerURL: false, hasWorkflow: false,
			wantError: true, errorMsg: validation.InvalidAccessTokenMsg,
		},
		"invalid server url": {
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "",
			hasWorkflow: false,
			wantError:   true, errorMsg: validation.InvalidServerURLMsg,
		},
		"no workflow": {
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "https://localhost:8080",
			hasWorkflow: false, wantError: false,
		},
		"invalid mandatory workflow": {
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: false, workflow: "",
			wantError: true, errorMsg: validation.InvalidWorkflowMsg,
		},
		"optional workflow": {
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: true,
			workflow: "", wantError: false,
		},
		"valid mandatory workflow": {
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: false,
			workflow: "workflow", wantError: false,
		},
		"all info": {
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "https://localhost:8080",
			hasWorkflow: true, isWorkflowOptional: false,
			workflow: "workflow", wantError: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := NewRootCmd()
			f := cmd.Flags()
			if test.hasToken {
				f.String("access-token", test.token, "")
			}
			if test.hasServerURL {
				viper.Set("server-url", test.serverURL)
				t.Cleanup(func() {
					viper.Reset()
				})
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
		})
	}
}

func TestSetupViper(t *testing.T) {
	tests := map[string]struct {
		env       string
		viperProp string
		value     string
	}{
		"server url": {
			env:       "REANA_SERVER_URL",
			viperProp: "server-url",
			value:     "https://localhost:8080",
		},
		"access token": {env: "REANA_ACCESS_TOKEN", viperProp: "access-token", value: "1234"},
		"workflow":     {env: "REANA_WORKON", viperProp: "workflow", value: "workflow"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(test.env, test.value)
			err := setupViper(nil)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				viper.Reset()
			})

			viperValue := viper.GetString(test.viperProp)
			if viperValue != test.value {
				t.Errorf(
					"Expected '%s' to be '%s', instead got '%s'",
					test.viperProp,
					test.value,
					viperValue,
				)
			}
		})
	}
}

func TestSetupLogger(t *testing.T) {
	tests := map[string]struct {
		level   string
		isValid bool
	}{
		"valid debug": {level: "DEBUG", isValid: true},
		"valid info":  {level: "INFO", isValid: true},
		"invalid":     {level: "QUIET", isValid: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
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
		})
	}
}
