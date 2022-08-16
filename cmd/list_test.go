/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"testing"

	"golang.org/x/exp/slices"
)

var listServerPath = "/api/workflows"

func TestBuildListHeader(t *testing.T) {
	tests := map[string]struct {
		runType              string
		verbose              bool
		includeWorkspaceSize bool
		includeProgress      bool
		includeDuration      bool
		expected             []string
	}{}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !slices.Equal(header, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, header)
			}
		})
	}
}
