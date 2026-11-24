/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"strings"
	"testing"

	"reanahub/reana-client-go/client/operations"
)

func TestWorkflowProgress(t *testing.T) {
	reported := &operations.GetWorkflowStatusOKBodyProgress{
		Total: &operations.GetWorkflowStatusOKBodyProgressTotal{Total: 2},
	}
	tests := []struct {
		name          string
		status        *operations.GetWorkflowStatusOKBody
		wantTotal     *operations.GetWorkflowStatusOKBodyProgressTotal
		wantErrorText string
	}{
		{
			name:          "empty response",
			wantErrorText: "workflow status response is empty",
		},
		{
			name:   "missing progress",
			status: &operations.GetWorkflowStatusOKBody{},
		},
		{
			name:      "reported progress",
			status:    &operations.GetWorkflowStatusOKBody{Progress: reported},
			wantTotal: reported.Total,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			progress, err := workflowProgress(test.status)
			if test.wantErrorText != "" {
				if err == nil ||
					!strings.Contains(err.Error(), test.wantErrorText) {
					t.Fatalf(
						"expected error %q, got %v",
						test.wantErrorText,
						err,
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if progress == nil {
				t.Fatal("expected a non-nil progress")
			}
			if progress.Total != test.wantTotal {
				t.Errorf(
					"unexpected total: got %v, want %v",
					progress.Total,
					test.wantTotal,
				)
			}
		})
	}
}
