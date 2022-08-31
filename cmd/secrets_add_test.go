/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/base64"
	"net/http"
	"os"
	"reanahub/reana-client-go/client/operations"
	"reflect"
	"testing"

	"golang.org/x/exp/slices"
)

var secretsAddServerPath = "/api/secrets/"

func TestSecretsAdd(t *testing.T) {
	emptyFile := t.TempDir() + "/empty.txt"
	_, err := os.Create(emptyFile)
	if err != nil {
		t.Fatalf("Error while creating empty file: %s", err.Error())
	}

	tests := map[string]TestCmdParams{
		"valid secrets": {
			serverResponses: map[string]ServerResponse{
				secretsAddServerPath: {
					statusCode:   http.StatusCreated,
					responseFile: "common_empty.json",
				},
			},
			args: []string{"--env", "PASSWORD=password", "--file", emptyFile},
			expected: []string{
				"Secrets PASSWORD, empty.txt were successfully uploaded",
			},
		},
		"unexisting file": {
			args:      []string{"--file", "invalid.txt"},
			wantError: true,
			expected: []string{
				"invalid value for '--file': file 'invalid.txt' does not exist",
			},
		},
		"invalid env secret": {
			args:      []string{"--env", "INVALID"},
			wantError: true,
			expected: []string{
				"Option \"INVALID\" is invalid:\nFor literal strings use \"SECRET_NAME=VALUE\" format",
			},
		},
		"no secrets": {
			wantError: true,
			expected: []string{
				"at least one of the options: 'env', 'file' is required",
				"Usage",
			},
		},
		"secret already exists": {
			serverResponses: map[string]ServerResponse{
				secretsAddServerPath: {
					statusCode:   http.StatusConflict,
					responseFile: "secrets_add_repeated.json",
				},
			},
			args:      []string{"--env", "PASSWORD=password"},
			wantError: true,
			expected: []string{
				"Operation cancelled. Secret PASSWORD already exists. If you want to change it use overwrite",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "secrets-add"
			testCmdRun(t, params)
		})
	}
}

func TestParseSecrets(t *testing.T) {
	tempDir := t.TempDir()
	emptyFile := tempDir + "/empty.txt"
	piFile := tempDir + "/pi.txt"
	_, err := os.Create(emptyFile)
	if err != nil {
		t.Fatalf("Error while creating empty file: %s", err.Error())
	}
	err = os.WriteFile(piFile, []byte("3.14"), 0777)
	if err != nil {
		t.Fatalf("Error while creating pi file: %s", err.Error())
	}

	tests := map[string]struct {
		envSecrets    []string
		fileSecrets   []string
		secrets       map[string]operations.AddSecretsParamsBodyAnon
		secretNames   []string
		wantError     bool
		expectedError string
	}{
		"no secrets": {
			secrets:     map[string]operations.AddSecretsParamsBodyAnon{},
			secretNames: []string{},
		},
		"env secret": {
			envSecrets: []string{"PASSWORD=password"},
			secrets: map[string]operations.AddSecretsParamsBodyAnon{
				"PASSWORD": {
					Name:  "PASSWORD",
					Type:  "env",
					Value: base64.StdEncoding.EncodeToString([]byte("password")),
				},
			},
			secretNames: []string{"PASSWORD"},
		},
		"file secret": {
			fileSecrets: []string{piFile},
			secrets: map[string]operations.AddSecretsParamsBodyAnon{
				"pi.txt": {
					Name:  "pi.txt",
					Type:  "file",
					Value: base64.StdEncoding.EncodeToString([]byte("3.14")),
				},
			},
			secretNames: []string{"pi.txt"},
		},
		"multiple secrets": {
			envSecrets:  []string{"PASSWORD=password", "USER=reanauser"},
			fileSecrets: []string{emptyFile, piFile},
			secrets: map[string]operations.AddSecretsParamsBodyAnon{
				"PASSWORD": {
					Name:  "PASSWORD",
					Type:  "env",
					Value: base64.StdEncoding.EncodeToString([]byte("password")),
				},
				"USER": {
					Name:  "USER",
					Type:  "env",
					Value: base64.StdEncoding.EncodeToString([]byte("reanauser")),
				},
				"empty.txt": {
					Name:  "empty.txt",
					Type:  "file",
					Value: "",
				},
				"pi.txt": {
					Name:  "pi.txt",
					Type:  "file",
					Value: base64.StdEncoding.EncodeToString([]byte("3.14")),
				},
			},
			secretNames: []string{"PASSWORD", "USER", "empty.txt", "pi.txt"},
		},
		"invalid env secret": {
			envSecrets:    []string{"INVALID"},
			wantError:     true,
			expectedError: "Option \"INVALID\" is invalid:\nFor literal strings use \"SECRET_NAME=VALUE\" format",
		},
		"invalid file secret": {
			fileSecrets:   []string{"invalid.txt"},
			wantError:     true,
			expectedError: "file invalid.txt could not be uploaded: open invalid.txt: no such file or directory",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			secrets, secretNames, err := parseSecrets(test.envSecrets, test.fileSecrets)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != test.expectedError {
					t.Errorf("Expected error: %s, got: %s", test.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err.Error())
				}
				if !reflect.DeepEqual(secrets, test.secrets) {
					t.Errorf("Expected: %v, got: %v", test.secrets, secrets)
				}
				if !slices.Equal(secretNames, test.secretNames) {
					t.Errorf("Expected: %v, got: %v", test.secretNames, secretNames)
				}
			}
		})
	}
}
