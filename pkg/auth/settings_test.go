/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsAndUnknownFieldsSurviveCredentialWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	t.Setenv(configPathEnv, path)
	t.Setenv(tlsVerifyEnv, "")
	t.Setenv(caCertsEnv, "")
	server := "https://reana.example.org"
	original := `{"version":3,"future":{"setting":true},"active_server":"https://reana.example.org","servers":{"https://reana.example.org":{"future_entry":42,"tls":{"verify":false,"future_tls":"keep"},"access_token":"old"}}}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &Store{Path: path}
	if _, err := store.Put(server, Credentials{AccessToken: "new"}, false); err != nil {
		t.Fatal(err)
	}
	verify, err := TLSVerify(server, nil)
	if err != nil || verify {
		t.Fatalf("saved policy = %v, %v", verify, err)
	}
	if _, err := store.ClearTokens(server, ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	entry := config["servers"].(map[string]any)[server].(map[string]any)
	if config["version"] != float64(3) || config["future"] == nil ||
		entry["future_entry"] != float64(42) ||
		entry["tls"].(map[string]any)["future_tls"] != "keep" {
		t.Fatalf("fields lost: %s", data)
	}
	if _, exists := entry["access_token"]; exists {
		t.Fatalf("logout retained a token: %s", data)
	}
}

func TestFailedLoginRecoveryDoesNotSelectServerOrSaveTLS(t *testing.T) {
	manager := testManager(t, nil)
	initial, target := "https://first.example.org", "https://second.example.org"
	verified, bypass := true, false
	if _, err := manager.Store.Put(initial, Credentials{}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Store.Put(target, Credentials{TLS: &TLSSettings{Verify: &verified}}, false); err != nil {
		t.Fatal(err)
	}
	manager.TLSVerify = &bypass
	_, err := manager.storeTokens(
		target,
		Metadata{},
		tokenResponse{RefreshToken: "rotated"},
		"",
		true,
	)
	if err == nil {
		t.Fatal("malformed response accepted")
	}
	active, _ := manager.Store.ActiveServer()
	entry, _ := manager.Store.Get(target)
	if active != initial || entry.RefreshToken != "rotated" ||
		!*entry.TLS.Verify {
		t.Fatalf(
			"failed login changed settings: active=%s entry=%+v",
			active,
			entry,
		)
	}
	if _, err = manager.storeTokens(target, Metadata{}, tokenResponse{AccessToken: "a.b.c"}, "", true); err != nil {
		t.Fatal(err)
	}
	entry, _ = manager.Store.Get(target)
	active, _ = manager.Store.ActiveServer()
	if active != target || *entry.TLS.Verify {
		t.Fatal("successful login did not save the requested policy")
	}
}

func TestCertificateErrorsKeepTypedCauseAndScopedAdvice(t *testing.T) {
	t.Setenv(caCertsEnv, "")
	server := "https://reana.example.org"
	cause := x509.UnknownAuthorityError{}
	err := ConnectionError(server, server+"/token?secret=hidden", cause)
	var authority x509.UnknownAuthorityError
	if !errors.As(err, &authority) ||
		!strings.Contains(err.Error(), "--no-tls-verify") ||
		strings.Contains(err.Error(), "hidden") {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
	err = ConnectionError(server, "https://iam.example.org/token", cause)
	if strings.Contains(err.Error(), "--no-tls-verify") {
		t.Fatal("suggested bypass for external IAM")
	}
	t.Setenv(caCertsEnv, "/trusted/ca.pem")
	bypass := false
	if _, err = TLSVerify(server, &bypass); err == nil {
		t.Fatal("accepted ineffective explicit bypass")
	}
	if status, err := TLSStatus(server, nil); err != nil ||
		status != "enabled" {
		t.Fatalf("effective CA policy: %s, %v", status, err)
	}
}
