/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/
package cmd

import (
	"testing"

	"reanahub/reana-client-go/pkg/auth"
)

func savedTestServer(t *testing.T, server string, verify bool) {
	t.Helper()
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/client.json")
	t.Setenv("REANA_SERVER_URL", "")
	t.Setenv("REANA_SERVER_TLS_VERIFY", "")
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(
		server,
		auth.Credentials{TLS: &auth.TLSSettings{Verify: &verify}},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
}
