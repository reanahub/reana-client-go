/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// writeSpec writes the given reana.yaml body and returns its path.
func writeSpec(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "reana.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeSerialSpec writes a minimal serial reana.yaml and returns its path.
func writeSerialSpec(t *testing.T) string {
	t.Helper()
	return writeSpec(t, "workflow:\n  type: serial\n")
}

func TestValidateKeepsPullLocal(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	var gotQuery url.Values
	var bundleError error
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			bundleError = checkBundleRequest(r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(
				[]byte(
					`{"valid": true, "errors": [], "warnings": [], "environments_truncated": true}`,
				),
			)
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	output, err := ExecuteCommand(
		NewRootCmd(),
		"validate", "-t", "1234", "-f", reanaFile, "--environments", "--pull",
	)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if !strings.Contains(output, "local environment checks were incomplete") {
		t.Fatalf("expected partial inspection result, got %q", output)
	}
	if gotQuery.Get("environments") != "true" {
		t.Errorf(
			"expected environments=true, got %q",
			gotQuery.Get("environments"),
		)
	}
	if gotQuery.Has("pull") {
		t.Errorf("expected pull to remain local, got %q", gotQuery.Get("pull"))
	}
	if bundleError != nil {
		t.Fatal(bundleError)
	}
}

func TestValidateOmitsEnvironmentFlagsByDefault(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	var gotQuery url.Values
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(
				[]byte(`{"valid": true, "errors": [], "warnings": []}`),
			)
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(),
		"validate",
		"-t",
		"1234",
		"-f",
		reanaFile,
	)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if gotQuery.Has("environments") || gotQuery.Has("pull") {
		t.Errorf("expected no environment flags, got %v", gotQuery)
	}
}

func TestValidateDisplaysRateLimitMessage(t *testing.T) {
	reanaFile := writeSerialSpec(t)
	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Please retry later."}`))
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(), "validate", "-t", "1234", "-f", reanaFile,
	)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if err.Error() != "Please retry later." {
		t.Fatalf("unexpected rate limit error: %v", err)
	}
}

func TestValidateDisplaysClientErrorMessages(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			reanaFile := writeSerialSpec(t)
			server := httptest.NewTLSServer(
				withPingFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write(
						[]byte(`{"message":"actionable validation error"}`),
					)
				}),
			)
			defer server.Close()

			savedTestServer(t, server.URL, false)
			viper.Set("server-url", server.URL)
			t.Cleanup(viper.Reset)

			_, err := ExecuteCommand(
				NewRootCmd(), "validate", "-t", "1234", "-f", reanaFile,
			)
			if err == nil {
				t.Fatal("expected validation request to fail")
			}
			if err.Error() != "actionable validation error" {
				t.Fatalf("unexpected client error: %v", err)
			}
		})
	}
}

func TestValidatePullRequiresEnvironments(t *testing.T) {
	reanaFile := writeSerialSpec(t)
	viper.Set("server-url", "https://localhost:1")
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(), "validate", "-t", "1234", "-f", reanaFile, "--pull",
	)
	if err == nil {
		t.Fatal("expected an error when --pull is used without --environments")
	}
}

func TestValidateWarnsServerCapabilitiesIsIgnored(t *testing.T) {
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(
				[]byte(`{"valid": true, "errors": [], "warnings": []}`),
			)
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"validate",
		"-t",
		"1234",
		"-f",
		reanaFile,
		"--server-capabilities",
	)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if !strings.Contains(out, "server-capabilities") {
		t.Errorf(
			"expected a warning that --server-capabilities is ignored, got %q",
			out,
		)
	}
}

func TestValidateReportsInvalidSpecification(t *testing.T) {
	// The command's primary purpose: a 200 report with valid=false must render
	// the errors and return a non-nil error (so the CLI exits non-zero).
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(
				[]byte(
					`{"valid": false, "errors": [{"message": "bad image"}], "warnings": []}`,
				),
			)
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	out, err := ExecuteCommand(
		NewRootCmd(), "validate", "-t", "1234", "-f", reanaFile,
	)
	if err == nil {
		t.Fatal("expected a non-nil error for an invalid specification")
	}
	if !strings.Contains(out, "bad image") {
		t.Errorf("expected the validation error to be displayed, got %q", out)
	}
}

func TestValidateSurfacesServerError(t *testing.T) {
	// A non-200 response is a server error, surfaced via report.Message rather
	// than being mislabeled as an invalid specification.
	reanaFile := writeSerialSpec(t)

	server := httptest.NewTLSServer(
		withPingFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(
				[]byte(`{"message": "validation service unavailable"}`),
			)
		}),
	)
	defer server.Close()

	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	_, err := ExecuteCommand(
		NewRootCmd(), "validate", "-t", "1234", "-f", reanaFile,
	)
	if err == nil {
		t.Fatal("expected a non-nil error for a server error response")
	}
	if !strings.Contains(err.Error(), "validation service unavailable") {
		t.Errorf("expected the server message to be surfaced, got %v", err)
	}
}

func TestContainsInt(t *testing.T) {
	if !containsInt([]int{0, 1, 2}, 0) {
		t.Error("expected 0 to be found")
	}
	if containsInt([]int{1, 2}, 0) {
		t.Error("did not expect 0 to be found")
	}
}

func TestCheckImagesLocallyNoImages(t *testing.T) {
	if got := checkImagesLocally(nil); got != nil {
		t.Errorf("expected nil for no images, got %v", got)
	}
	if got := checkImagesLocally([]imageEnvironment{{}, {}}); got != nil {
		t.Errorf("expected nil for only-empty images, got %v", got)
	}
}

func TestCheckImagesLocallyChecksSameImageUnderEachRuntimeUID(t *testing.T) {
	defaultUID, customUID, runtimeGID := 1000, 2000, 0
	environments := []imageEnvironment{
		{
			Image:      "busybox:1.36",
			RuntimeUID: defaultUID,
			RuntimeGID: runtimeGID,
		},
		{
			Image:      "busybox:1.36",
			RuntimeUID: customUID,
			RuntimeGID: runtimeGID,
		},
	}

	originalFindCLI := findLocalContainerCLI
	originalInspect := inspectImageUIDGIDs
	t.Cleanup(func() {
		findLocalContainerCLI = originalFindCLI
		inspectImageUIDGIDs = originalInspect
	})
	findLocalContainerCLI = func() string { return "docker" }
	inspectionCalls := 0
	inspectImageUIDGIDs = func(_ string, _ string) (int, []int, error) {
		inspectionCalls++
		return customUID, []int{runtimeGID}, nil
	}

	messages := checkImagesLocally(environments)
	if len(messages) != 1 {
		t.Fatalf("expected one UID warning, got %v", messages)
	}
	if !strings.Contains(messages[0], "UID 1000") {
		t.Errorf("expected warning for the default UID, got %q", messages[0])
	}
	if inspectionCalls != 1 {
		t.Errorf("expected one image inspection, got %d", inspectionCalls)
	}
}

func TestLocalContainerCLI(t *testing.T) {
	root := t.TempDir()
	podmanDir := filepath.Join(root, "podman")
	dockerDir := filepath.Join(root, "docker")
	if err := os.MkdirAll(podmanDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dockerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(podmanDir, "podman"), "#!/bin/sh\n")
	t.Setenv("PATH", podmanDir)
	if got := localContainerCLI(); got != "podman" {
		t.Errorf("expected podman fallback, got %q", got)
	}

	writeExecutable(t, filepath.Join(dockerDir, "docker"), "#!/bin/sh\n")
	t.Setenv("PATH", dockerDir+string(os.PathListSeparator)+podmanDir)
	if got := localContainerCLI(); got != "docker" {
		t.Errorf("expected docker to be preferred, got %q", got)
	}

	t.Setenv("PATH", t.TempDir())
	if got := localContainerCLI(); got != "" {
		t.Errorf("expected no container CLI, got %q", got)
	}
}

func writeExecutable(t *testing.T, name, contents string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test requires POSIX executable scripts")
	}
	if err := os.WriteFile(name, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestImageUIDGIDs(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		script    string
		uid       int
		gids      []int
		errorText string
	}{
		{
			name: "pull failure is ignored",
			script: `#!/bin/sh
if [ "$1" = "pull" ]; then exit 1; fi
printf '1000\n0 100 invalid\n'
`,
			uid:  1000,
			gids: []int{0, 100},
		},
		{
			name: "unexpected output",
			script: `#!/bin/sh
if [ "$1" = "run" ]; then printf '1000\n'; fi
`,
			errorText: "unexpected id output",
		},
		{
			name: "invalid uid",
			script: `#!/bin/sh
if [ "$1" = "run" ]; then printf 'user\n0\n'; fi
`,
			errorText: `could not parse uid "user"`,
		},
		{
			name: "runtime error includes stderr",
			script: `#!/bin/sh
if [ "$1" = "run" ]; then echo 'image unavailable' >&2; exit 2; fi
`,
			errorText: "image unavailable",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cli := filepath.Join(t.TempDir(), "container-cli")
			writeExecutable(t, cli, testCase.script)
			uid, gids, err := imageUIDGIDs(cli, "example/image:latest")
			if testCase.errorText != "" {
				if err == nil ||
					!strings.Contains(err.Error(), testCase.errorText) {
					t.Fatalf(
						"expected %q error, got uid=%d gids=%v err=%v",
						testCase.errorText,
						uid,
						gids,
						err,
					)
				}
				return
			}
			if err != nil || uid != testCase.uid ||
				len(gids) != len(testCase.gids) {
				t.Fatalf("unexpected uid=%d gids=%v err=%v", uid, gids, err)
			}
			for index := range gids {
				if gids[index] != testCase.gids[index] {
					t.Fatalf("unexpected gids %v", gids)
				}
			}
		})
	}
}

func TestImageUIDGIDsTimeoutForcesNamedCleanup(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "calls")
	cli := filepath.Join(t.TempDir(), "container-cli")
	writeExecutable(t, cli, fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
if [ "$1" = "run" ]; then sleep 2; fi
`, logPath))
	originalTimeout := localImageCheckTimeout
	localImageCheckTimeout = time.Second
	defer func() { localImageCheckTimeout = originalTimeout }()

	if _, _, err := imageUIDGIDs(cli, "example/image:latest"); err == nil {
		t.Fatal("expected inspection timeout")
	}
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
	var runName string
	var cleanupName string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 2 && fields[0] == "run" {
			for index, field := range fields {
				if field == "--name" && index+1 < len(fields) {
					runName = fields[index+1]
				}
			}
		}
		if len(fields) == 3 && fields[0] == "rm" && fields[1] == "-f" {
			cleanupName = fields[2]
		}
	}
	if runName == "" || cleanupName != runName {
		t.Fatalf("expected cleanup of %q, calls were %q", runName, contents)
	}
}

func TestCheckImagesLocallyWarnings(t *testing.T) {
	originalFindCLI := findLocalContainerCLI
	originalInspect := inspectImageUIDGIDs
	t.Cleanup(func() {
		findLocalContainerCLI = originalFindCLI
		inspectImageUIDGIDs = originalInspect
	})

	findLocalContainerCLI = func() string { return "" }
	messages := checkImagesLocally([]imageEnvironment{{Image: "busybox"}})
	if len(messages) != 1 || !strings.Contains(messages[0], "were skipped") {
		t.Fatalf("expected skipped warning, got %v", messages)
	}

	findLocalContainerCLI = func() string { return "docker" }
	inspectionCalls := 0
	inspectImageUIDGIDs = func(_ string, _ string) (int, []int, error) {
		inspectionCalls++
		return 0, nil, errors.New("inspect failed")
	}
	environments := []imageEnvironment{
		{Image: "busybox", RuntimeUID: 1000, RuntimeGID: 1000},
		{Image: "busybox", RuntimeUID: 2000, RuntimeGID: 2000},
	}
	messages = checkImagesLocally(environments)
	if len(messages) != 1 || !strings.Contains(messages[0], "inspect failed") {
		t.Fatalf("expected one inspection warning, got %v", messages)
	}
	if inspectionCalls != 1 {
		t.Errorf("expected one cached inspection, got %d", inspectionCalls)
	}

	inspectImageUIDGIDs = func(_ string, _ string) (int, []int, error) {
		return 1000, []int{100}, nil
	}
	messages = checkImagesLocally([]imageEnvironment{
		{Image: "busybox", RuntimeUID: 2000, RuntimeGID: 200},
		{Image: "busybox", RuntimeUID: 2000, RuntimeGID: 200},
	})
	if len(messages) != 2 ||
		!strings.Contains(messages[0], "GID 200") ||
		!strings.Contains(messages[1], "UID is 1000") {
		t.Fatalf("expected GID and UID warnings, got %v", messages)
	}

	messages = checkImagesLocally([]imageEnvironment{
		{Image: "busybox", RuntimeUID: 1000, RuntimeGID: 100},
	})
	if len(messages) != 0 {
		t.Fatalf("expected matching image identity, got %v", messages)
	}
}
