/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"testing"
)

var uploadServerPath = "/api/workflows/%s/workspace"

func TestFileUpload(t *testing.T) {
	testFile := t.TempDir() + "/test.txt"
	_, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Error while creating test file: %s", err.Error())
	}

	tests := map[string]TestCmdParams{
		"valid upload": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(uploadServerPath, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "upload_success.json",
				},
			},
			args: []string{"-w", "my_workflow", testFile},
			expected: []string{
				"test.txt was successfully uploaded.",
			},
		},
		"unexisting file": {
			args:      []string{"-w", "my_workflow", "non_existing"},
			wantError: true,
			expected: []string{
				"no such file or directory",
			},
		},
		"unexisting workflow": {
			args:      []string{},
			wantError: true,
			expected: []string{
				"workflow name must be provided",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "upload"
			testCmdRun(t, params)
		})
	}
}
