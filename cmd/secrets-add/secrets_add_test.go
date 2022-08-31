/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package secrets_add

import (
	"encoding/base64"
	"net/http"
	"os"
	"reanahub/reana-client-go/client/operations"
	. "reanahub/reana-client-go/cmd/internal"
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
			ServerResponses: map[string]ServerResponse{
				secretsAddServerPath: {
					StatusCode:   http.StatusCreated,
					ResponseFile: "../../testdata/inputs/empty.json",
				},
			},
			Args: []string{"--env", "PASSWORD=password", "--file", emptyFile},
			Expected: []string{
				"Secrets PASSWORD, empty.txt were successfully uploaded",
			},
		},
		"unexisting file": {
			Args:      []string{"--file", "invalid.txt"},
			WantError: true,
			Expected: []string{
				"invalid value for '--file': file 'invalid.txt' does not exist",
			},
		},
		"invalid env secret": {
			Args:      []string{"--env", "INVALID"},
			WantError: true,
			Expected: []string{
				"Option \"INVALID\" is invalid:\nFor literal strings use \"SECRET_NAME=VALUE\" format",
			},
		},
		"no secrets": {
			WantError: true,
			Expected: []string{
				"at least one of the options: 'env', 'file' is required",
				"Usage",
			},
		},
		"secret already exists": {
			ServerResponses: map[string]ServerResponse{
				secretsAddServerPath: {
					StatusCode:   http.StatusConflict,
					ResponseFile: "testdata/repeated.json",
				},
			},
			Args:      []string{"--env", "PASSWORD=password"},
			WantError: true,
			Expected: []string{
				"Operation cancelled. Secret PASSWORD already exists. If you want to change it use overwrite",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.Cmd = NewCmd()
			TestCmdRun(t, params)
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
