/*
This file is part of REANA.
Copyright (C) 2022, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"strings"
	"testing"

	"github.com/reanahub/reana-client-go/pkg/auth"
)

func TestVersion(t *testing.T) {
	cmd := newVersionCmd()
	out, _ := ExecuteCommand(cmd)

	if strings.TrimSpace(out) != version {
		t.Fatalf("Expected: \"%s\", got: \"%s\"", version, out)
	}
}

func TestVersionIsReportedToAuth(t *testing.T) {
	if auth.ClientVersion != version {
		t.Fatalf(
			"auth.ClientVersion = %q, want %q",
			auth.ClientVersion,
			version,
		)
	}
}
