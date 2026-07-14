/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
)

func captureTLSWarning(t *testing.T) *bytes.Buffer {
	t.Helper()
	output := &bytes.Buffer{}
	logger := log.StandardLogger()
	previousOutput := logger.Out
	logger.SetOutput(output)
	tlsWarningOnce = sync.Once{}
	t.Cleanup(func() {
		logger.SetOutput(previousOutput)
		tlsWarningOnce = sync.Once{}
	})
	return output
}

func TestNewHTTPClientWarnsOnceWhenVerificationDisabled(t *testing.T) {
	output := captureTLSWarning(t)
	t.Setenv(tlsVerifyEnv, "false")
	t.Setenv(caCertsEnv, "")

	if _, err := NewHTTPClient(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHTTPClient(); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(output.String(), "disabled by REANA_SERVER_TLS_VERIFY"); count != 1 {
		t.Fatalf("warning count = %d, output = %q", count, output.String())
	}
}

func TestNewHTTPClientRejectsInvalidTLSVerifyValue(t *testing.T) {
	output := captureTLSWarning(t)
	t.Setenv(tlsVerifyEnv, "banana")
	t.Setenv(caCertsEnv, "")

	_, err := NewHTTPClient()
	if err == nil || !strings.Contains(err.Error(), "invalid") ||
		!strings.Contains(err.Error(), tlsVerifyEnv) {
		t.Fatalf("expected invalid %s error, got %v", tlsVerifyEnv, err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid setting warned: %q", output.String())
	}
}

func TestParseBoolean(t *testing.T) {
	cases := []struct {
		value string
		want  bool
		ok    bool
	}{
		{"1", true, true},
		{"true", true, true},
		{"True", true, true},
		{"yes", true, true},
		{"on", true, true},
		{"0", false, true},
		{"false", false, true},
		{"no", false, true},
		{"off", false, true},
		{"  true  ", true, true},
		{"banana", false, false},
		{"", false, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.value, func(t *testing.T) {
			got, ok := parseBoolean(testCase.value)
			if got != testCase.want || ok != testCase.ok {
				t.Fatalf(
					"parseBoolean(%q) = (%v, %v), want (%v, %v)",
					testCase.value,
					got,
					ok,
					testCase.want,
					testCase.ok,
				)
			}
		})
	}
}

func TestDefaultConfigPathErrorsWhenHomeIsUnset(t *testing.T) {
	t.Setenv("HOME", "")
	if _, err := defaultConfigPath(); err == nil {
		t.Fatal("expected an error when the home directory cannot be resolved")
	}
}

func TestConfigPathReturnsAbsolutePathFromEnv(t *testing.T) {
	t.Setenv(configPathEnv, "relative/credentials.json")
	got, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs("relative/credentials.json")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestConfigPathExpandsTildePrefix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(
		configPathEnv,
		"~"+string(filepath.Separator)+"custom"+string(
			filepath.Separator,
		)+"credentials.json",
	)
	got, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "custom", "credentials.json")
	if got != want {
		t.Fatalf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestConfigPathTildeExpansionErrorsWhenHomeIsUnset(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv(
		configPathEnv,
		"~"+string(filepath.Separator)+"credentials.json",
	)
	if _, err := ConfigPath(); err == nil {
		t.Fatal("expected an error when the home directory cannot be resolved")
	}
}

func TestNewHTTPClientKeepsVerificationAndCABundlePrecedenceSilent(
	t *testing.T,
) {
	output := captureTLSWarning(t)
	t.Setenv(tlsVerifyEnv, "true")
	if _, err := NewHTTPClient(); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("enabled verification warned: %q", output.String())
	}

	tlsWarningOnce = sync.Once{}
	t.Setenv(tlsVerifyEnv, "false")
	caPath := t.TempDir() + "/invalid-ca.pem"
	if err := os.WriteFile(caPath, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(caCertsEnv, caPath)
	if _, err := NewHTTPClient(); err == nil {
		t.Fatal("invalid CA bundle unexpectedly accepted")
	}
	if output.Len() != 0 {
		t.Fatalf("CA precedence emitted insecure warning: %q", output.String())
	}
}
