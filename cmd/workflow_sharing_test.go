/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"golang.org/x/exp/slices"
)

func TestWorkflowSharingMultiUserResults(t *testing.T) {
	workflowName := "my_workflow"
	users := []string{"alice@cern.ch", "fail@cern.ch", "carol@cern.ch"}
	shareMessage := "Please review my analysis"
	validUntil := "2099-12-31"

	tests := map[string]struct {
		command       string
		path          string
		successField  string
		successFormat string
		action        string
	}{
		"share add": {
			command:       "share-add",
			path:          fmt.Sprintf(shareAddPathTemplate, workflowName),
			successField:  "shared_with",
			successFormat: workflowName + " is now read-only shared with %s",
			action:        "share",
		},
		"share remove": {
			command:       "share-remove",
			path:          fmt.Sprintf(shareRemovePathTemplate, workflowName),
			successField:  "unshared_with",
			successFormat: workflowName + " is no longer shared with %s",
			action:        "unshare",
		},
	}
	results := map[string]map[string]bool{
		"success": {},
		"partial failure": {
			users[1]: true,
		},
		"full failure": {
			users[0]: true,
			users[1]: true,
			users[2]: true,
		},
	}

	for name, test := range tests {
		for resultName, failedUsers := range results {
			for _, jsonOutput := range []bool{false, true} {
				mode := "text"
				if jsonOutput {
					mode = "json"
				}
				t.Run(name+" "+resultName+" "+mode, func(t *testing.T) {
					var requestedUsers []string
					server := httptest.NewTLSServer(http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							if r.URL.Path != test.path {
								t.Errorf("unexpected request to %q", r.URL.Path)
								http.NotFound(w, r)
								return
							}
							if got := r.Header.Get("Authorization"); got != "Bearer 1234" {
								t.Errorf(
									"expected Authorization Bearer 1234, got %q",
									got,
								)
							}
							if r.URL.Query().Has("access_token") {
								t.Errorf(
									"access token leaked into query: %s",
									r.URL.RawQuery,
								)
							}

							user := r.URL.Query().
								Get("user_email_to_unshare_with")
							if test.command == "share-add" {
								var body struct {
									User       *string `json:"user_email_to_share_with"`
									Message    string  `json:"message"`
									ValidUntil string  `json:"valid_until"`
								}
								if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
									t.Errorf(
										"could not decode share request: %v",
										err,
									)
									http.Error(
										w,
										"invalid body",
										http.StatusBadRequest,
									)
									return
								}
								if body.User != nil {
									user = *body.User
								}
								if body.Message != shareMessage {
									t.Errorf(
										"expected message %q, got %q",
										shareMessage,
										body.Message,
									)
								}
								if body.ValidUntil != validUntil {
									t.Errorf(
										"expected valid-until %q, got %q",
										validUntil,
										body.ValidUntil,
									)
								}
							}
							requestedUsers = append(requestedUsers, user)

							w.Header().Set("Content-Type", "application/json")
							if failedUsers[user] {
								w.WriteHeader(http.StatusNotFound)
								_, _ = fmt.Fprint(
									w,
									`{"message":"sharing request failed"}`,
								)
								return
							}
							_, _ = fmt.Fprint(w, `{}`)
						},
					))
					viper.Set("server-url", server.URL)
					t.Cleanup(func() {
						server.Close()
						viper.Reset()
					})

					args := []string{
						test.command,
						"-t", "1234",
						"-w", workflowName,
					}
					for _, user := range users {
						args = append(args, "--user", user)
					}
					if test.command == "share-add" {
						args = append(
							args,
							"--message", shareMessage,
							"--valid-until", validUntil,
						)
					}
					if jsonOutput {
						args = append(args, "--json")
					}

					output, err := ExecuteCommand(NewRootCmd(), args...)
					if len(failedUsers) > 0 &&
						!errors.Is(err, config.ErrEmpty) {
						t.Fatalf("expected sharing failure, got %v", err)
					}
					if len(failedUsers) == 0 && err != nil {
						t.Fatalf("expected sharing success, got %v", err)
					}
					if !slices.Equal(requestedUsers, users) {
						t.Fatalf(
							"expected requests for %v, got %v",
							users,
							requestedUsers,
						)
					}

					successfulUsers := make([]string, 0, len(users))
					for _, user := range users {
						if !failedUsers[user] {
							successfulUsers = append(successfulUsers, user)
						}
					}

					if jsonOutput {
						assertSharingJSON(
							t,
							output,
							workflowName,
							test.successField,
							successfulUsers,
							failedUsers,
						)
					} else {
						if len(successfulUsers) > 0 {
							expected := fmt.Sprintf(
								test.successFormat,
								strings.Join(successfulUsers, ", "),
							)
							if !strings.Contains(output, expected) {
								t.Errorf("expected %q in output %q", expected, output)
							}
						} else if strings.Contains(output, "SUCCESS") {
							t.Errorf("unexpected success output %q", output)
						}
						for user := range failedUsers {
							expected := fmt.Sprintf(
								"Failed to %s %s with %s: sharing request failed",
								test.action,
								workflowName,
								user,
							)
							if !strings.Contains(output, expected) {
								t.Errorf("expected %q in output %q", expected, output)
							}
						}
					}
				})
			}
		}
	}
}

func assertSharingJSON(
	t *testing.T,
	output, expectedWorkflow, successField string,
	expectedUsers []string,
	failedUsers map[string]bool,
) {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatalf("expected valid JSON output, got %q: %v", output, err)
	}
	var workflow string
	if err := json.Unmarshal(raw["workflow"], &workflow); err != nil {
		t.Fatalf("could not decode workflow: %v", err)
	}
	if workflow != expectedWorkflow {
		t.Errorf("expected workflow %q, got %q", expectedWorkflow, workflow)
	}
	var actualUsers []string
	if err := json.Unmarshal(raw[successField], &actualUsers); err != nil {
		t.Fatalf("could not decode %s: %v", successField, err)
	}
	if !slices.Equal(actualUsers, expectedUsers) {
		t.Errorf(
			"expected successful users %v, got %v",
			expectedUsers,
			actualUsers,
		)
	}
	var sharingErrors []string
	if err := json.Unmarshal(raw["errors"], &sharingErrors); err != nil {
		t.Fatalf("could not decode errors: %v", err)
	}
	if len(sharingErrors) != len(failedUsers) {
		t.Fatalf(
			"expected %d sharing errors, got %v",
			len(failedUsers),
			sharingErrors,
		)
	}
	for user := range failedUsers {
		found := false
		for _, sharingError := range sharingErrors {
			if strings.Contains(sharingError, user) &&
				strings.Contains(sharingError, "sharing request failed") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(
				"expected sharing error for %s, got %v",
				user,
				sharingErrors,
			)
		}
	}
}
