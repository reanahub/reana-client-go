/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/reanahub/reana-client-go/client/operations"
)

func TestWorkflowInputs(t *testing.T) {
	inputs := &operations.GetWorkflowSpecificationOKBodySpecificationInputs{
		Files:       []string{"data/input.txt"},
		Directories: []string{"data/events"},
	}
	tests := []struct {
		name          string
		spec          *operations.GetWorkflowSpecificationOKBody
		wantFiles     []string
		wantDirs      []string
		wantErrorText string
	}{
		{
			name:          "empty response",
			wantErrorText: "workflow specification response is empty",
		},
		{
			name:          "missing specification",
			spec:          &operations.GetWorkflowSpecificationOKBody{},
			wantErrorText: "workflow specification response is missing specification",
		},
		{
			name: "missing inputs",
			spec: &operations.GetWorkflowSpecificationOKBody{
				Specification: &operations.GetWorkflowSpecificationOKBodySpecification{},
			},
		},
		{
			name: "declared inputs",
			spec: &operations.GetWorkflowSpecificationOKBody{
				Specification: &operations.GetWorkflowSpecificationOKBodySpecification{
					Inputs: inputs,
				},
			},
			wantFiles: inputs.Files,
			wantDirs:  inputs.Directories,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			files, dirs, err := workflowInputs(test.spec)
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
			if !reflect.DeepEqual(files, test.wantFiles) {
				t.Errorf(
					"unexpected files: got %v, want %v",
					files,
					test.wantFiles,
				)
			}
			if !reflect.DeepEqual(dirs, test.wantDirs) {
				t.Errorf(
					"unexpected directories: got %v, want %v",
					dirs,
					test.wantDirs,
				)
			}
		})
	}
}
