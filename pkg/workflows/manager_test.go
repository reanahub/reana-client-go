/*
This file is part of REANA.
Copyright (C) 2022, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"reanahub/reana-client-go/client/operations"

	"github.com/spf13/viper"
)

func useWorkflowServer(t *testing.T, handler http.Handler) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	viper.Set("server-url", server.URL)
	t.Cleanup(func() {
		viper.Reset()
		server.Close()
	})
}

func TestWorkflowOperations(t *testing.T) {
	dir := t.TempDir()
	localFile := filepath.Join(dir, "local.txt")
	if err := os.WriteFile(localFile, []byte("file contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	var requests []string
	useWorkflowServer(
		t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method+" "+r.URL.Path)
			if got := r.URL.Query().Get("access_token"); got != "token" {
				t.Errorf("expected access token, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodPut &&
				strings.HasSuffix(r.URL.Path, "/status"):
				if got := r.URL.Query().Get("status"); got != "deleted" {
					t.Errorf("expected deleted status, got %q", got)
				}
				var body map[string]bool
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if !body["workspace"] || !body["all_runs"] {
					t.Errorf("unexpected status body %#v", body)
				}
				_, _ = w.Write([]byte(`{"message":"updated"}`))
			case r.Method == http.MethodGet &&
				strings.HasSuffix(r.URL.Path, "/status"):
				_, _ = w.Write([]byte(`{"name":"analysis","status":"running"}`))
			case strings.HasSuffix(r.URL.Path, "/specification"):
				_, _ = w.Write([]byte(`{"parameters":{"events":10}}`))
			case r.Method == http.MethodPost &&
				strings.HasSuffix(r.URL.Path, "/workspace"):
				assertUploadedFile(
					t,
					r,
					"canonical/reana.yaml",
					"file contents",
				)
				_, _ = w.Write([]byte(`{"message":"uploaded"}`))
			case r.Method == http.MethodGet &&
				strings.Contains(r.URL.Path, "/workspace/"):
				w.Header().Set(
					"Content-Disposition",
					`attachment; filename="results.zip"`,
				)
				w.Header().Set("Content-Type", "application/zip")
				_, _ = w.Write([]byte("zip contents"))
			default:
				http.NotFound(w, r)
			}
		}),
	)

	if err := UpdateStatus("token", "analysis", "deleted", true, true); err != nil {
		t.Fatal(err)
	}
	status, err := GetStatus("token", "analysis")
	if err != nil {
		t.Fatal(err)
	}
	if status.Name != "analysis" || status.Status != "running" {
		t.Errorf("unexpected status payload %#v", status)
	}
	specification, err := GetWorkflowSpecification("token", "analysis")
	if err != nil {
		t.Fatal(err)
	}
	parameters, ok := specification.Parameters.(map[string]interface{})
	if !ok || fmt.Sprint(parameters["events"]) != "10" {
		t.Errorf("unexpected specification payload %#v", specification)
	}
	message, err := UploadFileAs(
		"token", "analysis", localFile, "canonical/reana.yaml",
	)
	if err != nil {
		t.Fatal(err)
	}
	if message != "uploaded" {
		t.Errorf("unexpected upload response %q", message)
	}
	name, contents, zipped, err := DownloadFile(
		"token", "analysis", "results/*.txt",
	)
	if err != nil {
		t.Fatal(err)
	}
	if name != "results.zip" || contents.String() != "zip contents" || !zipped {
		t.Errorf(
			"unexpected download name=%q contents=%q zipped=%t",
			name,
			contents.String(),
			zipped,
		)
	}
	if len(requests) != 5 {
		t.Errorf("expected five API requests, got %v", requests)
	}
}

func assertUploadedFile(
	t *testing.T,
	r *http.Request,
	workspaceName, expectedContents string,
) {
	t.Helper()
	if got := r.URL.Query().Get("file_name"); got != workspaceName {
		t.Errorf("expected workspace name %q, got %q", workspaceName, got)
	}
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		t.Errorf("unexpected content type %q", r.Header.Get("Content-Type"))
	}
	if r.ContentLength != int64(len(expectedContents)) {
		t.Errorf("unexpected content length %d", r.ContentLength)
	}
	contents, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != expectedContents {
		t.Errorf("unexpected uploaded contents %q", contents)
	}
}

func TestUploadFileUsesLocalName(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("input.txt", []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	useWorkflowServer(
		t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assertUploadedFile(t, r, "input.txt", "input")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"uploaded"}`))
		}),
	)

	if _, err := UploadFile("token", "analysis", "input.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestUploadEmptyFileHasKnownZeroLength(t *testing.T) {
	localFile := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(localFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength != 0 {
				t.Errorf(
					"expected zero content length, got %d",
					r.ContentLength,
				)
			}
			if len(r.TransferEncoding) != 0 {
				t.Errorf(
					"empty upload used transfer encoding %v",
					r.TransferEncoding,
				)
			}
			contents, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if len(contents) != 0 {
				t.Errorf("expected an empty body, got %q", contents)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"uploaded"}`))
		}),
	)
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := uploadFileAs(
		server.Client(), serverURL, "token", "analysis", localFile, "empty",
	); err != nil {
		t.Fatal(err)
	}
}

func TestUploadWorkflowNameIsEscapedOnce(t *testing.T) {
	tests := []struct {
		workflow    string
		escapedPath string
	}{
		{"analysis space", "/api/workflows/analysis%20space/workspace"},
		{"analysis#fragment", "/api/workflows/analysis%23fragment/workspace"},
		{"analysis?query", "/api/workflows/analysis%3Fquery/workspace"},
		{"analysis%percent", "/api/workflows/analysis%25percent/workspace"},
	}
	for _, test := range tests {
		t.Run(test.workflow, func(t *testing.T) {
			localFile := filepath.Join(t.TempDir(), "input")
			if err := os.WriteFile(localFile, []byte("input"), 0o600); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if got := r.URL.EscapedPath(); got != test.escapedPath {
						t.Errorf(
							"expected escaped path %q, got %q",
							test.escapedPath,
							got,
						)
					}
					assertUploadedFile(t, r, "input.txt", "input")
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"message":"uploaded"}`))
				}),
			)
			defer server.Close()
			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := uploadFileAs(
				server.Client(), serverURL, "token", test.workflow, localFile, "input.txt",
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDownloadFileDefaultsFilename(t *testing.T) {
	useWorkflowServer(
		t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Disposition", "attachment")
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("result"))
		}),
	)

	name, contents, zipped, err := DownloadFile(
		"token", "analysis", "result.txt",
	)
	if err != nil {
		t.Fatal(err)
	}
	if name != "downloaded_file" || contents.String() != "result" || zipped {
		t.Errorf(
			"unexpected download name=%q contents=%q zipped=%t",
			name,
			contents.String(),
			zipped,
		)
	}
}

func TestDownloadFileAllowsLargeResponses(t *testing.T) {
	// File downloads are not subject to validation's control-response cap.
	payload := bytes.Repeat([]byte("x"), 16*1024*1024+1)
	for _, testCase := range []struct {
		name          string
		contentLength bool
	}{
		{"known length", true},
		{"chunked", false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			useWorkflowServer(
				t,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Disposition", "attachment")
					w.Header().Set("Content-Type", "application/octet-stream")
					if testCase.contentLength {
						w.Header().Set(
							"Content-Length",
							strconv.Itoa(len(payload)),
						)
					} else {
						w.WriteHeader(http.StatusOK)
						w.(http.Flusher).Flush()
					}
					_, _ = w.Write(payload)
				}),
			)

			_, contents, _, err := DownloadFile(
				"token", "analysis", "large-result.bin",
			)
			if err != nil {
				t.Fatal(err)
			}
			if contents.Len() != len(payload) {
				t.Fatalf(
					"expected %d downloaded bytes, got %d",
					len(payload),
					contents.Len(),
				)
			}
		})
	}
}

func TestWorkflowOperationErrors(t *testing.T) {
	t.Run("invalid status", func(t *testing.T) {
		err := UpdateStatus("token", "workflow", "invalid", false, false)
		if err == nil {
			t.Fatal("expected invalid status error")
		}
	})

	t.Run("missing upload file", func(t *testing.T) {
		_, err := UploadFile(
			"token",
			"workflow",
			filepath.Join(t.TempDir(), "missing"),
		)
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("expected missing file error, got %v", err)
		}
	})

	t.Run("server responses", func(t *testing.T) {
		useWorkflowServer(
			t,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"server failed"}`))
			}),
		)
		file := filepath.Join(t.TempDir(), "input.txt")
		if err := os.WriteFile(file, []byte("input"), 0o600); err != nil {
			t.Fatal(err)
		}
		checks := []struct {
			name string
			call func() error
		}{
			{"update", func() error {
				return UpdateStatus(
					"token",
					"workflow",
					"deleted",
					false,
					false,
				)
			}},
			{"status", func() error {
				_, err := GetStatus("token", "workflow")
				return err
			}},
			{"specification", func() error {
				_, err := GetWorkflowSpecification("token", "workflow")
				return err
			}},
			{"upload", func() error {
				_, err := UploadFile("token", "workflow", file)
				return err
			}},
			{"download", func() error {
				_, _, _, err := DownloadFile(
					"token", "workflow", "result.txt",
				)
				return err
			}},
		}
		for _, check := range checks {
			t.Run(check.name, func(t *testing.T) {
				if err := check.call(); err == nil {
					t.Errorf("expected %s server error", check.name)
				}
			})
		}
	})

	t.Run("malformed content disposition", func(t *testing.T) {
		useWorkflowServer(
			t,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Disposition", `"unterminated`)
				_, _ = fmt.Fprint(w, "result")
			}),
		)
		_, _, _, err := DownloadFile("token", "workflow", "result.txt")
		if err == nil {
			t.Fatal("expected malformed Content-Disposition error")
		}
	})
}

func TestInputAndOutputPaths(t *testing.T) {
	declared := &operations.GetWorkflowSpecificationOKBody{
		Specification: &operations.GetWorkflowSpecificationOKBodySpecification{
			Inputs: &operations.GetWorkflowSpecificationOKBodySpecificationInputs{
				Files:       []string{"data.txt"},
				Directories: []string{"code"},
			},
			Outputs: &operations.GetWorkflowSpecificationOKBodySpecificationOutputs{
				Files: []string{"results/plot.png"},
			},
		},
	}

	tests := map[string]struct {
		spec       *operations.GetWorkflowSpecificationOKBody
		wantInFile []string
		wantInDir  []string
		wantOutRes []string
	}{
		"nil payload": {spec: nil},
		"no specification": {
			spec: &operations.GetWorkflowSpecificationOKBody{},
		},
		"no inputs or outputs": {
			spec: &operations.GetWorkflowSpecificationOKBody{
				Specification: &operations.GetWorkflowSpecificationOKBodySpecification{},
			},
		},
		"both declared": {
			spec:       declared,
			wantInFile: []string{"data.txt"},
			wantInDir:  []string{"code"},
			wantOutRes: []string{"results/plot.png"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			files, directories := InputPaths(test.spec)
			if !reflect.DeepEqual(files, test.wantInFile) {
				t.Errorf("input files: got %v, want %v", files, test.wantInFile)
			}
			if !reflect.DeepEqual(directories, test.wantInDir) {
				t.Errorf(
					"input directories: got %v, want %v",
					directories,
					test.wantInDir,
				)
			}
			outFiles, outDirs := OutputPaths(test.spec)
			if !reflect.DeepEqual(outFiles, test.wantOutRes) {
				t.Errorf(
					"output files: got %v, want %v",
					outFiles,
					test.wantOutRes,
				)
			}
			if outDirs != nil {
				t.Errorf("output directories: got %v, want nil", outDirs)
			}
		})
	}
}
