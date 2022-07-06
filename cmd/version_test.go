/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"reanahub/reana-client-go/utils"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	cmd := NewRootCmd()
	out, _ := utils.ExecuteCommand(cmd, "version")

	if strings.TrimSpace(out) != version {
		t.Fatalf("Expected: \"%s\", got: \"%s\"", version, out)
	}
}
