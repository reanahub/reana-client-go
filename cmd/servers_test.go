/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"os"
	"strings"
	"testing"

	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/viper"
)

func serverCommand(t *testing.T, args ...string) string {
	t.Helper()
	viper.Reset()
	output, err := ExecuteCommand(NewRootCmd(), args...)
	if err != nil {
		t.Fatalf("%v: %v (%s)", args, err, output)
	}
	return output
}

func TestSavedServerCommands(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_URL", "")
	t.Setenv("REANA_SERVER_TLS_VERIFY", "")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	if output := serverCommand(t, "server-list"); !strings.Contains(
		output,
		"No saved REANA servers",
	) {
		t.Fatal(output)
	}
	serverCommand(t, "server-add", "B.EXAMPLE/", "--no-tls-verify")
	serverCommand(t, "server-add", "a.example")
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	active, err := store.ActiveServer()
	if err != nil || active != "" {
		t.Fatalf("unexpected selection: %s, %v", active, err)
	}
	entry, _ := store.Get("b.example")
	if entry.TLS == nil || entry.TLS.Verify == nil || *entry.TLS.Verify {
		t.Fatal("bypass not saved")
	}
	entry.AccessToken = "access-secret"
	entry.RefreshToken = "refresh-secret"
	if _, err := store.Put("b.example", entry, false); err != nil {
		t.Fatal(err)
	}
	serverCommand(t, "server-use", "B.EXAMPLE/")
	output := serverCommand(t, "server-list")
	if !strings.Contains(
		output,
		"* https://b.example  TLS verification: disabled",
	) ||
		strings.Contains(output, "secret") {
		t.Fatal(output)
	}
	if strings.Index(output, "a.example") > strings.Index(output, "b.example") {
		t.Fatal(output)
	}
	serverCommand(t, "server-use", "a.example")
	after, _ := store.Get("b.example")
	if after.RefreshToken != entry.RefreshToken ||
		after.AccessToken != entry.AccessToken {
		t.Fatal("selection changed credentials")
	}
	serverCommand(t, "server-remove", "b.example", "--local-only")
	active, _ = store.ActiveServer()
	if active != "https://a.example" {
		t.Fatal("removing another server changed selection")
	}
	serverCommand(t, "server-remove", "a.example")
	active, _ = store.ActiveServer()
	servers, _ := store.ListServers()
	if active != "" || len(servers) != 0 {
		t.Fatal("active removal did not clear selection")
	}
}

func TestServerCommandErrorsPreserveStore(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	serverCommand(t, "server-add", "a.example")
	serverCommand(t, "server-use", "a.example")
	before, _ := os.ReadFile(os.Getenv("REANA_CLIENT_CONFIG"))
	for _, args := range [][]string{
		{"server-add", "a.example", "--no-tls-verify"},
		{"server-use", "missing.example"},
		{"server-remove", "missing.example"},
		{"server-add", "b.example", "--tls-verify", "--no-tls-verify"},
		{"server-add", "http://remote.example"},
		{"server-add"}, {"server-use"}, {"server-remove"}, {"server-list", "extra"},
	} {
		viper.Reset()
		if _, err := ExecuteCommand(NewRootCmd(), args...); err == nil {
			t.Fatalf("expected failure: %v", args)
		}
		after, _ := os.ReadFile(os.Getenv("REANA_CLIENT_CONFIG"))
		if string(before) != string(after) {
			t.Fatalf("%v changed store", args)
		}
	}
}

func TestServerCommandsCABundle(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	serverCommand(t, "server-add", "a.example", "--no-tls-verify")
	t.Setenv("REANA_SERVER_CA_CERTS", "bundle.pem")
	if output := serverCommand(t, "server-list"); !strings.Contains(
		output,
		"TLS verification: enabled",
	) {
		t.Fatal(output)
	}
	_, err := ExecuteCommand(
		NewRootCmd(),
		"server-add",
		"b.example",
		"--no-tls-verify",
	)
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected bundle conflict: %v", err)
	}
}

func TestServerCommandsIgnoreRetiredExports(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_URL", "https://wrong.example")
	t.Setenv("REANA_SERVER_TLS_VERIFY", "0")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	serverCommand(t, "server-add", "a.example")
	serverCommand(t, "server-use", "a.example")
	output := serverCommand(t, "server-list")
	if !strings.Contains(
		output,
		"* https://a.example  TLS verification: enabled",
	) {
		t.Fatal(output)
	}
	if _, err := ExecuteCommand(NewRootCmd(), "ping"); err == nil ||
		!strings.Contains(err.Error(), "no longer client inputs") {
		t.Fatalf("ping should reject exports: %v", err)
	}
	serverCommand(t, "server-remove", "a.example")
	data, _ := os.ReadFile(os.Getenv("REANA_CLIENT_CONFIG"))
	if !strings.Contains(string(data), `"active_server": null`) {
		t.Fatal(string(data))
	}
}

func TestServerAddPreservesImplicitTLSDefault(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	serverCommand(t, "server-add", "a.example")
	serverCommand(t, "server-add", "b.example", "--tls-verify")
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	implicit, _ := store.Get("a.example")
	explicit, _ := store.Get("b.example")
	if implicit.TLS != nil || explicit.TLS == nil ||
		explicit.TLS.Verify == nil ||
		!*explicit.TLS.Verify {
		t.Fatal("implicit and explicit TLS choices lost")
	}
	output := serverCommand(t, "server-remove", "a.example", "--local-only")
	if strings.Contains(output, "not revoked") {
		t.Fatal(output)
	}
}
