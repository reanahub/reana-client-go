/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package version

import (
	. "reanahub/reana-client-go/cmd/internal"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	cmd := NewCmd()
	out, _ := ExecuteCommand(cmd)

	if strings.TrimSpace(out) != config.Version {
		t.Fatalf("Expected: \"%s\", got: \"%s\"", config.Version, out)
	}
}
