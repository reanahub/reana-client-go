/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package config

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestGetRunStatuses(t *testing.T) {
	tests := map[string]struct {
		includeDeleted bool
		numStatuses    int
	}{
		"exclude deleted": {includeDeleted: false, numStatuses: 7},
		"include deleted": {includeDeleted: true, numStatuses: 8},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runStatuses := GetRunStatuses(test.includeDeleted)
			if len(runStatuses) != test.numStatuses {
				t.Errorf("Expected %d statuses, got %d", test.numStatuses, len(runStatuses))
			}

			if test.includeDeleted {
				if !slices.Contains(runStatuses, "deleted") {
					t.Errorf("Expected runStatuses to contain deleted")
				}
			} else if slices.Contains(runStatuses, "deleted") {
				t.Errorf("Expected runStatuses not to contain deleted")
			}
		})
	}
}
