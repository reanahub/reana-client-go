/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/
package auth

import (
	"os"
	"testing"
)

// TestMain keeps a developer's stale export from changing the login tests.
func TestMain(m *testing.M) {
	if err := os.Unsetenv(retiredLoopbackPortEnv); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func savedTestServer(t *testing.T, server string, verify bool) {
	t.Helper()
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/client.json")
	t.Setenv("REANA_SERVER_URL", "")
	t.Setenv("REANA_SERVER_TLS_VERIFY", "")
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(
		server,
		Credentials{TLS: &TLSSettings{Verify: &verify}},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
}
