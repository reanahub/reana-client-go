/*
This file is part of REANA.
Copyright (C) 2022, 2024, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"reanahub/reana-client-go/pkg/auth"
	"reanahub/reana-client-go/pkg/errorhandler"
	"reanahub/reana-client-go/pkg/validator"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

// ExecuteCommand executes a cobra command with the given args.
// Returns the output of the command and any error it may provide.
func ExecuteCommand(
	cmd *cobra.Command,
	args ...string,
) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.String(), errorhandler.HandleApiError(err)
}

type TestCmdParams struct {
	cmd             string
	serverResponses map[string]ServerResponse
	args            []string
	expected        []string
	unwanted        []string
	wantError       bool
	serverURL       string
}

type ServerResponse struct {
	statusCode              int
	responseFile            string
	responseHeaders         map[string]string
	expectedQueryParams     map[string]string
	additionalResponseFiles []string
}

func TestRootCommandGroups(t *testing.T) {
	output, err := ExecuteCommand(NewRootCmd(), "--help")
	if err != nil {
		t.Fatalf("Got unexpected error '%s'", err.Error())
	}

	managementStart := strings.Index(
		output,
		"Workflow management commands:",
	)
	executionStart := strings.Index(
		output,
		"Workflow execution commands:",
	)
	sharingStart := strings.Index(output, "Workflow sharing commands:")
	if managementStart == -1 || executionStart == -1 || sharingStart == -1 ||
		managementStart >= executionStart || executionStart >= sharingStart {
		t.Fatalf("Expected workflow command groups in output, got '%s'", output)
	}

	managementCommands := output[managementStart:executionStart]
	if !strings.Contains(managementCommands, "\n  create ") {
		t.Errorf(
			"Expected create in workflow management commands, got '%s'",
			managementCommands,
		)
	}

	executionCommands := output[executionStart:sharingStart]
	if strings.Contains(executionCommands, "\n  create ") {
		t.Errorf(
			"Did not expect create in workflow execution commands, got '%s'",
			executionCommands,
		)
	}
}

// getResponseFile returns the response file for the given call number,
// allowing for additional response files to be used for the same endpoint,
// i. e. when the endpoint is called multiple times.
func getResponseFile(callSeqNum int, serverResponse ServerResponse) string {
	if len(serverResponse.additionalResponseFiles) == 0 {
		return serverResponse.responseFile
	}

	if callSeqNum == 0 {
		return serverResponse.responseFile
	}

	if callSeqNum < len(serverResponse.additionalResponseFiles)+1 {
		return serverResponse.additionalResponseFiles[callSeqNum-1]
	}

	return serverResponse.responseFile
}

func testCmdRun(t *testing.T, p TestCmdParams) {
	callSeqNum := 0
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authorization := r.Header.Get("Authorization"); authorization != "Bearer 1234" {
				t.Errorf(
					"Expected Authorization 'Bearer 1234', got '%v'",
					authorization,
				)
			}
			if r.URL.Query().Has("access_token") {
				t.Errorf("Access token leaked into query: %s", r.URL.RawQuery)
			}
			if strings.HasSuffix(r.URL.Path, "/interactive-session-secret") {
				w.Header().Add("Content-Type", "application/json")
				_, _ = w.Write(
					[]byte(
						`{"session_secret":"session-secret","path":"/session"}`,
					),
				)
				return
			}
			res, validPath := p.serverResponses[r.URL.Path]
			if validPath {
				for name, expected := range res.expectedQueryParams {
					if got := r.URL.Query().Get(name); got != expected {
						t.Errorf(
							"Expected query parameter '%s' to be '%s', got '%s'",
							name,
							expected,
							got,
						)
					}
				}
				w.Header().Add("Content-Type", "application/json")
				for name, value := range res.responseHeaders {
					w.Header().Add(name, value)
				}
				w.WriteHeader(res.statusCode)

				var body []byte
				responseFile := getResponseFile(callSeqNum, res)
				callSeqNum++
				if responseFile != "" {
					var err error
					body, err = os.ReadFile(
						"../testdata/inputs/" + responseFile,
					)
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
		}),
	)

	viper.Set("server-url", server.URL)
	if p.serverURL != "" {
		viper.Set("server-url", p.serverURL)
	}
	t.Cleanup(func() {
		server.Close()
		viper.Reset()
	})

	rootCmd := NewRootCmd()
	args := append([]string{p.cmd, "-t", "1234"}, p.args...)
	output, err := ExecuteCommand(rootCmd, args...)

	if !p.wantError && err != nil {
		t.Fatalf("Got unexpected error '%s'", err.Error())
	}
	if p.wantError && err == nil {
		t.Fatalf("Expected error, instead got '%s'", output)
	}

	for _, test := range p.expected {
		if !p.wantError && !strings.Contains(output, test) {
			t.Errorf("Expected '%s' in output, instead got '%s'", test, output)
		}
		if p.wantError && !strings.Contains(err.Error(), test) &&
			!strings.Contains(output, test) {
			t.Errorf(
				"Expected '%s' in error output, instead got '%s'",
				test,
				err.Error(),
			)
		}
	}

	for _, forbidden := range p.unwanted {
		if !p.wantError && strings.Contains(output, forbidden) {
			t.Errorf(
				"Expected '%s' not to be in output, instead got '%s'",
				forbidden,
				output,
			)
		}
		if p.wantError && (strings.Contains(err.Error(), forbidden) ||
			strings.Contains(output, forbidden)) {
			t.Errorf(
				"Expected '%s' not to be in error output, instead got '%s'",
				forbidden,
				err.Error(),
			)
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
			wantError: true, errorMsg: "REANA client is not connected to any REANA cluster; run `reana-client-go login`",
		},
		"invalid server url": {
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "",
			hasWorkflow: false,
			wantError:   true, errorMsg: validator.InvalidServerURLMsg,
		},
		"no workflow": {
			hasToken: true, token: "token",
			hasServerURL: true, serverURL: "https://localhost:8080",
			hasWorkflow: false, wantError: false,
		},
		"invalid mandatory workflow": {
			hasToken: false, hasServerURL: false,
			hasWorkflow: true, isWorkflowOptional: false, workflow: "",
			wantError: true, errorMsg: validator.InvalidWorkflowMsg,
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
			t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
			t.Setenv("REANA_SERVER_URL", "")
			t.Setenv("REANA_ACCESS_TOKEN", "")
			viper.Reset()
			t.Cleanup(viper.Reset)
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
					err := f.SetAnnotation(
						"workflow",
						"properties",
						[]string{"optional"},
					)
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
		"access token": {
			env:       "REANA_ACCESS_TOKEN",
			viperProp: "access-token",
			value:     "1234",
		},
		"workflow": {
			env:       "REANA_WORKON",
			viperProp: "workflow",
			value:     "workflow",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(test.env, test.value)
			err := setupViper()
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

func TestMalformedEnvironmentAccessTokenIsRejected(t *testing.T) {
	t.Setenv("REANA_ACCESS_TOKEN", "legacy-opaque-token")
	t.Setenv("REANA_SERVER_URL", "https://reana.example.org")
	viper.Reset()
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(NewRootCmd(), "ping")
	if err == nil || !strings.Contains(err.Error(), "must contain a JWT") {
		t.Fatalf("expected JWT migration error, got %v", err)
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

func TestLogCmdFlagsRedactsAccessToken(t *testing.T) {
	oldLevel := log.GetLevel()
	oldOutput := log.StandardLogger().Out
	log.SetLevel(log.DebugLevel)
	var output bytes.Buffer
	log.SetOutput(&output)
	t.Cleanup(func() {
		log.SetLevel(oldLevel)
		log.SetOutput(oldOutput)
	})

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("access-token", "", "")
	cmd.Flags().String("workflow", "", "")
	if err := cmd.Flags().Set("access-token", "secret-jwt"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("workflow", "analysis"); err != nil {
		t.Fatal(err)
	}

	logCmdFlags(cmd)
	logged := output.String()
	if strings.Contains(logged, "secret-jwt") {
		t.Fatalf("access token was logged: %s", logged)
	}
	if !strings.Contains(logged, "access-token: [REDACTED]") {
		t.Fatalf("redaction marker missing: %s", logged)
	}
	if !strings.Contains(logged, "workflow: analysis") {
		t.Fatalf("non-sensitive flag missing: %s", logged)
	}
}

func TestValidateFlagsResolvesServerURLFromActiveCredentialStore(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_URL", "")
	t.Setenv("REANA_ACCESS_TOKEN", "")
	viper.Reset()
	t.Cleanup(viper.Reset)

	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put("https://reana.example.org", auth.Credentials{
		AccessToken: "a.b.c",
		AccessTokenExpiresAt: time.Now().
			Add(time.Hour).
			Format(time.RFC3339),
	}, true); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.Flags().String("access-token", "", "")

	if err := validateFlags(cmd); err != nil {
		t.Fatalf("Got unexpected error '%s'", err.Error())
	}
	if got := viper.GetString("server-url"); got != "https://reana.example.org" {
		t.Fatalf(
			"server-url = %q, want the active credential store server",
			got,
		)
	}
}

func TestBindViperToCmdFlagPropagatesSetError(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("retries", "not-a-number")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Int("retries", 0, "")
	flag := flags.Lookup("retries")

	if err := bindViperToCmdFlag(flag); err == nil {
		t.Fatal(
			"expected an error when the viper value cannot be applied to the flag",
		)
	}
}
