/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package tester

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reanahub/reana-client-go/pkg/auth"
	"reflect"
	"strings"
	"testing"

	"reanahub/reana-client-go/client"

	"github.com/spf13/viper"
)

func newTestAPIFetcher(
	t *testing.T,
	handler http.HandlerFunc,
) *APIFetcher {
	t.Helper()
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/client.json")
	t.Setenv("REANA_SERVER_TLS_VERIFY", "")
	t.Setenv("REANA_SERVER_URL", "")
	server := httptest.NewTLSServer(handler)
	viper.Set("server-url", server.URL)
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	verify := false
	if _, err := store.Put(server.URL, auth.Credentials{TLS: &auth.TLSSettings{Verify: &verify}}, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		server.Close()
		viper.Reset()
	})
	api, err := client.ApiClient("1234")
	if err != nil {
		t.Fatal(err)
	}
	return NewAPIFetcher(api)
}

func TestAPIFetcherMapsWorkflowResponses(t *testing.T) {
	fetcher := newTestAPIFetcher(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer 1234" {
				t.Errorf("got Authorization %q, want Bearer 1234", got)
			}
			if r.URL.Query().Has("access_token") {
				t.Errorf("access token leaked into query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/workflows/analysis.1/status":
				_, _ = w.Write([]byte(`{
                "name":"analysis.1",
                "status":"finished",
                "progress":{
                  "run_started_at":"2026-08-23T09:00:00",
                  "run_finished_at":"2026-08-23T09:05:00"
                }
              }`))
			case "/api/workflows/analysis.1/specification":
				_, _ = w.Write([]byte(`{
                "parameters":{},
                "specification":{
                  "outputs":{"files":["result.txt"],"directories":["plots"]},
                  "tests":{"files":["tests/workflow.feature"]}
                }
              }`))
			case "/api/workflows/analysis.1/workspace":
				if name := r.URL.Query().Get("file_name"); name != "result.txt" {
					t.Errorf("got file_name %q, want result.txt", name)
				}
				_, _ = w.Write([]byte(`{
                "items":[{
                  "name":"result.txt",
                  "size":{"raw":12,"human_readable":"12 B"}
                }],
                "total":1
              }`))
			case "/api/workflows/analysis.1/disk_usage":
				var body struct {
					Summarize bool `json:"summarize"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("could not decode disk usage request: %v", err)
				}
				if !body.Summarize {
					t.Error("disk usage request did not set summarize")
				}
				_, _ = w.Write([]byte(`{
                "disk_usage_info":[{
                  "name":"",
                  "size":{"raw":12,"human_readable":"12 B"}
                }]
              }`))
			case "/api/workflows/analysis.1/logs":
				var steps []string
				if err := json.NewDecoder(r.Body).Decode(&steps); err != nil {
					t.Errorf("could not decode logs request: %v", err)
				}
				if !reflect.DeepEqual(
					steps,
					[]string{"step1"},
				) {
					t.Errorf("got steps %v, want [step1]", steps)
				}
				_, _ = w.Write([]byte(`{
                "logs":"{\"workflow_logs\":\"engine\",\"job_logs\":{\"1\":{\"job_name\":\"step1\",\"logs\":\"job\"}},\"engine_specific\":{\"dag\":{\"nodes\":[]}}}"
              }`))
			default:
				if r.URL.Path == "/api/workflows/analysis.1/workspace/result.txt" {
					w.Header().Set("Content-Type", "application/octet-stream")
					_, _ = w.Write([]byte("hello world"))
					return
				}
				t.Fatalf("unexpected request path %s", r.URL.Path)
			}
		},
	)

	status, err := fetcher.Status("analysis.1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Name != "analysis.1" || status.Status != "finished" ||
		status.RunFinishedAt != "2026-08-23T09:05:00" {
		t.Errorf("unexpected status: %+v", status)
	}

	specification, err := fetcher.Specification("analysis.1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(
		specification.TestFiles,
		[]string{"tests/workflow.feature"},
	) || !reflect.DeepEqual(
		specification.OutputDirectories,
		[]string{"plots"},
	) {
		t.Errorf("unexpected specification: %+v", specification)
	}

	files, err := fetcher.Files("analysis.1", "result.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Size != 12 {
		t.Errorf("unexpected files: %+v", files)
	}

	diskUsage, err := fetcher.DiskUsage("analysis.1", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diskUsage) != 1 || diskUsage[0].Size != 12 {
		t.Errorf("unexpected disk usage: %+v", diskUsage)
	}

	logs, err := fetcher.Logs("analysis.1", []string{"step1"})
	if err != nil {
		t.Fatal(err)
	}
	if logs.WorkflowLogs != "engine" || logs.JobLogs["1"].Logs != "job" ||
		!strings.Contains(engineSpecificText(logs.EngineSpecific), `"dag"`) {
		t.Errorf("unexpected logs: %+v", logs)
	}

	download, err := fetcher.Download("analysis.1", "result.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(download.Content) != "hello world" || download.IsArchive {
		t.Errorf("unexpected download: %+v", download)
	}
}

func TestAPIFetcherContentAssertionsUseRelativePaths(t *testing.T) {
	fetcher := newTestAPIFetcher(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			const path = "/api/workflows/analysis.1/workspace/results/message.txt"
			if r.URL.Path != path {
				t.Errorf("got download path %q, want %q", r.URL.Path, path)
				http.Redirect(w, r, path, http.StatusPermanentRedirect)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("hello world"))
		},
	)

	for _, filename := range []string{
		"results/message.txt",
		`"results/message.txt"`,
		`"/results/message.txt"`,
	} {
		t.Run(filename, func(t *testing.T) {
			if err := assertFileContains(fetcher)(
				"analysis.1",
				map[string]string{"filename": filename, "content": "hello world"},
			); err != nil {
				t.Fatalf("file content assertion failed: %v", err)
			}
			if err := assertFileChecksum(fetcher)(
				"analysis.1",
				map[string]string{
					"filename":  filename,
					"algorithm": "sha256",
					"checksum":  "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
				},
			); err != nil {
				t.Fatalf("checksum assertion failed: %v", err)
			}
		})
	}
}
