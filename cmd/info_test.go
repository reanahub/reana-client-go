/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"testing"
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
			},
			expected: []string{
				"List of supported compute backends: kubernetes, slurmcern",
				"Default timeout for Kubernetes jobs: 124",
				"Default memory limit for Kubernetes jobs: 248",
				"Default workspace: /var/reana",
				"Maximum allowed memory limit for Kubernetes jobs: 1000",
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
			},
		},
		"json": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_big.json",
				},
			},
			args: []string{"--json"},
			expected: []string{
				"\"compute_backends\": {", "\"value\": [", "\"kubernetes\",", "\"slurmcern\"",
				"\"default_kubernetes_jobs_timeout\": {", "\"value\": \"124\"",
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
			},
		},
		"dask": {
			serverResponses: map[string]ServerResponse{
				infoServerPath: {
					statusCode:   http.StatusOK,
					responseFile: "info_dask.json",
				},
			},
			expected: []string{
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
			},
			expected: []string{
				"Maximum allowed memory limit for Kubernetes jobs: None",
				"Maximum retention period in days for workspace files: None",
			},
			unwanted: []string{
				"List of supported compute backends", "Default workspace",
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
