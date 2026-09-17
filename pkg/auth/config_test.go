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
	savedTestServer(t, "https://reana.example.org", false)
	t.Setenv(caCertsEnv, "")

	if _, err := NewHTTPClient("https://reana.example.org"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHTTPClient("https://reana.example.org"); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(output.String(), "disabled for"); count != 1 {
		t.Fatalf("warning count = %d, output = %q", count, output.String())
	}
}

func TestNewHTTPClientRejectsRetiredTLSVariable(t *testing.T) {
	output := captureTLSWarning(t)
	t.Setenv(tlsVerifyEnv, "banana")
	t.Setenv(caCertsEnv, "")

	_, err := NewHTTPClient("https://reana.example.org")
	if err == nil ||
		!strings.Contains(err.Error(), "no longer client inputs") ||
		!strings.Contains(err.Error(), tlsVerifyEnv) {
		t.Fatalf("expected invalid %s error, got %v", tlsVerifyEnv, err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid setting warned: %q", output.String())
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
	savedTestServer(t, "https://reana.example.org", true)
	if _, err := NewHTTPClient("https://reana.example.org"); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("enabled verification warned: %q", output.String())
	}

	tlsWarningOnce = sync.Once{}
	savedTestServer(t, "https://reana.example.org", false)
	caPath := t.TempDir() + "/invalid-ca.pem"
	if err := os.WriteFile(caPath, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(caCertsEnv, caPath)
	if _, err := NewHTTPClient("https://reana.example.org"); err == nil {
		t.Fatal("invalid CA bundle unexpectedly accepted")
	}
	if output.Len() != 0 {
		t.Fatalf("CA precedence emitted insecure warning: %q", output.String())
	}
}
