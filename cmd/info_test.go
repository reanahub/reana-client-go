/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2024, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var infoServerPath = "/api/info"

func TestInfo(t *testing.T) {
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_big.json",
				},
				quotaShowServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "quota_show_complete.json",
				},
			},
			expected: []string{
				"List of supported compute backends: kubernetes, slurmcern",
				"CPU quota period in months: 3",
				"Current CPU quota period start: 2026-06-04",
				"Current CPU quota period end: 2026-09-04",
				"Default timeout for Kubernetes jobs: 124",
				"Default workspace: /var/reana",
				"GitLab host: gitlab.cern.ch",
				"Default CPU request for Kubernetes jobs: 1",
				"Maximum allowed CPU request for Kubernetes jobs: 2",
				"Default CPU limit for Kubernetes jobs: 2",
				"Maximum allowed CPU limit for Kubernetes jobs: 4",
				"Default memory request for Kubernetes jobs: 1Gi",
				"Maximum allowed memory request for Kubernetes jobs: 5Gi",
				"Default memory limit for Kubernetes jobs: 3Gi",
				"Maximum allowed memory limit for Kubernetes jobs: 10Gi",
				"Maximum inactivity period in days before automatic closure of interactive sessions: 7",
				"Maximum timeout for Kubernetes jobs: 500",
				"Maximum retention period in days for workspace files: 250",
				"List of available workspaces: /var/reana, /var/cern",
				"Users can set custom interactive session images: False",
				"Recommended jupyter images for interactive sessions: docker.io/jupyter/scipy-notebook:notebook-6.4.5",
				"List of supported workflow engines: cwl, serial, snakemake, yadage",
				"CWL engine tool: cwltool",
				"CWL engine version: 3.1.20210628163208",
				"Yadage engine version: 0.20.1",
				"Yadage engine adage version: 0.11.0",
				"Yadage engine packtivity version: 0.16.2",
				"Snakemake engine version: 8.24.1",
				"Dask workflows allowed in the cluster: False",
			},
		},
		"json": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_big.json",
				},
				quotaShowServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "quota_show_complete.json",
				},
			},
			args: []string{"--json"},
			expected: []string{
				"\"compute_backends\": {", "\"value\": [", "\"kubernetes\",", "\"slurmcern\"",
				"\"cpu_quota_period_months\": {", "\"value\": 3",
				"\"current_cpu_quota_period_start\": {", "\"value\": \"2026-06-04\"",
				"\"current_cpu_quota_period_end\": {", "\"value\": \"2026-09-04\"",
				"\"default_kubernetes_jobs_timeout\": {", "\"value\": \"124\"",
				"\"gitlab_host\": {", "\"value\": \"gitlab.cern.ch\"",
				"\"workspaces_available\": {", "\"value\": [", "\"/var/reana\",", "\"/var/cern\"",
				"\"interactive_sessions_custom_image_allowed\": {", "\"value\": \"False\"",
				"\"interactive_session_recommended_jupyter_images\": {", "\"value\": [", "\"docker.io/jupyter/scipy-notebook:notebook-6.4.5\"",
				"\"supported_workflow_engines\": {", "\"value\": [", "\"cwl\",", "\"serial\",", "\"snakemake\",", "\"yadage\"",
				"\"cwl_engine_tool\": {", "\"value\": \"cwltool\"",
				"\"cwl_engine_version\": {", "\"value\": \"3.1.20210628163208\"",
				"\"yadage_engine_version\": {", "\"value\": \"0.20.1\"",
				"\"yadage_engine_adage_version\": {", "\"value\": \"0.11.0\"",
				"\"yadage_engine_packtivity_version\": {", "\"value\": \"0.16.2\"",
				"\"snakemake_engine_version\": {", "\"value\": \"8.24.1\"",
				"\"dask_enabled\": {", "\"value\": \"False\"",
			},
		},
		"dask": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_dask.json",
				},
				quotaShowServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "quota_show_complete.json",
				},
			},
			expected: []string{
				"CPU quota period in months: 3",
				"Current CPU quota period start: 2026-06-04",
				"Current CPU quota period end: 2026-09-04",
				"Dask workflows allowed in the cluster: True",
				"Dask autoscaler enabled in the cluster: True",
				"The number of Dask workers created by default: 2",
				"The maximum memory limit for Dask clusters created by users: 16Gi",
				"The amount of memory used by default by a single Dask worker: 2Gi",
				"The maximum amount of memory that users can ask for the single Dask worker: 8Gi",
				"The maximum number of workers that users can ask for the single Dask cluster: 20",
			},
		},
		"missing fields": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_small.json",
				},
				quotaShowServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "quota_show_no_info.json",
				},
			},
			expected: []string{
				"CPU quota period in months: 0",
				"Maximum allowed memory limit for Kubernetes jobs: None",
				"Maximum retention period in days for workspace files: None",
			},
			unwanted: []string{
				"List of supported compute backends",
				"Default workspace",
				"Current CPU quota period start:",
				"Current CPU quota period end:",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "info"
			testCmdRun(t, params)
		})
	}
}

func TestInfoQuotaPeriodIsPrintedAfterInfoBlock(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if accessToken := r.URL.Query().Get("access_token"); accessToken != "1234" {
				t.Errorf("Expected access token '1234', got '%v'", accessToken)
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			var responseFile string
			switch r.URL.Path {
			case infoServerPath:
				responseFile = "../testdata/inputs/info_small.json"
			case quotaShowServerPath:
				responseFile = "../testdata/inputs/quota_show_no_info.json"
			default:
				t.Fatalf("Unexpected request to '%v'", r.URL.Path)
			}

			body, err := os.ReadFile(responseFile)
			if err != nil {
				t.Fatalf("Error while reading response file: %v", err)
			}
			if _, err := w.Write(body); err != nil {
				t.Fatalf("Error while writing response body: %v", err)
			}
		}),
	)

	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		server.Close()
		viper.Reset()
	})

	rootCmd := NewRootCmd()
	output, err := ExecuteCommand(rootCmd, "info", "-t", "1234")
	if err != nil {
		t.Fatalf("Got unexpected error '%v'", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	infoIdx := -1
	quotaIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(
			trimmed,
			"Maximum allowed memory limit for Kubernetes jobs:",
		) {
			infoIdx = i
		}
		if strings.HasPrefix(trimmed, "CPU quota period in months:") {
			quotaIdx = i
		}
	}
	if infoIdx < 0 || quotaIdx < 0 || quotaIdx <= infoIdx {
		t.Fatalf(
			"Expected CPU quota period to be printed after the info block, got order info=%d quota=%d in:\n%s",
			infoIdx,
			quotaIdx,
			output,
		)
	}
}
