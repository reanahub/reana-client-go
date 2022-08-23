/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"fmt"
	"reanahub/reana-client-go/pkg/config"
	"strings"
	"testing"
)

func TestUpdateStatus(t *testing.T) {
	err := UpdateStatus("token", "workflow", "invalid", false, false)
	if err == nil {
		t.Errorf("expected %s error, got nil", fmt.Errorf(
			"invalid value for status: invalid is not part of '%s'",
			strings.Join(config.GetRunStatuses(true), "', '"),
		))
	}
}
