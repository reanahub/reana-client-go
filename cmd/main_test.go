// This file is part of REANA.
// Copyright (C) 2026 CERN.

package cmd

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := os.Setenv("REANA_SERVER_TLS_VERIFY", "false"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
