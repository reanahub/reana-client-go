// This file is part of REANA.
// Copyright (C) 2026 CERN.
//
// REANA is free software; you can redistribute it and/or modify it
// under the terms of the MIT License; see LICENSE file for more details.

package cmd

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "reana-command-tests-")
	if err != nil {
		panic(err)
	}
	for _, name := range []string{"REANA_SERVER_URL", "REANA_SERVER_TLS_VERIFY", "REANA_SERVER_CA_CERTS", "REANA_ACCESS_TOKEN"} {
		if err := os.Unsetenv(name); err != nil {
			panic(err)
		}
	}
	if err := os.Setenv("REANA_CLIENT_CONFIG", directory+"/client.json"); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(directory)
	os.Exit(code)
}
