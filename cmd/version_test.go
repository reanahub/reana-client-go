/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	cmd := newVersionCmd()
	out, _ := ExecuteCommand(cmd)

	if strings.TrimSpace(out) != version {
		t.Fatalf("Expected: \"%s\", got: \"%s\"", version, out)
	}
}
