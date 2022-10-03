/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
)

var downloadWorkflowSpecServerPath = "/api/workflows/%s/specification"
var downloadServerPath = "/api/workflows/%s/workspace/%s"

func TestFileDownload(t *testing.T) {

	fileName := "results/plot.png"
	dirName := "results"
	dirZipFileName := "download_roofit.1_results_2022-10-03-122917.zip"

	tests := map[string]TestCmdParams{
		"download file specified in the workflow specification as outputs": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadWorkflowSpecServerPath, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "workflow_specification.json",
				},
				fmt.Sprintf(downloadServerPath, "my_workflow", fileName): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, fileName),
					},
				},
			},
			args: []string{"-w", "my_workflow"},
			expected: []string{
				fmt.Sprintf("%s was successfully downloaded.", fileName),
			},
		},
		"download file specified as argument": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadServerPath, "my_workflow", fileName): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, fileName),
					},
				},
			},
			args: []string{"-w", "my_workflow", fileName},
			expected: []string{
				fmt.Sprintf("%s was successfully downloaded.", fileName),
			},
		},
		"download directory specified as argument": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadServerPath, "my_workflow", dirName): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
					responseHeaders: map[string]string{
						"Content-Disposition": fmt.Sprintf(
							`attachment; filename="%s"`,
							dirZipFileName,
						),
					},
				},
			},
			args: []string{"-w", "my_workflow", dirName, "-o", dirName},
			expected: []string{
				fmt.Sprintf("%s was successfully downloaded.", dirZipFileName),
			},
		},
		"download unexisting file": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(downloadServerPath, "my_workflow", "file"): {
					statusCode:   http.StatusNotFound,
					responseFile: "download_file_not_found.json",
				},
			},
			args:      []string{"-w", "my_workflow", "file"},
			wantError: true,
			expected: []string{
				"file does not exist.",
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
			params.cmd = "download"
			testCmdRun(t, params)
		})
	}
	t.Cleanup(func() {
		// Remove all the temp files created by the test
		err := os.RemoveAll(dirName)
		if err != nil {
			log.Fatal(err)
		}
	})
}
