/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/reanahub/reana-client-go/client"
	"github.com/reanahub/reana-client-go/pkg/config"
	"github.com/reanahub/reana-client-go/pkg/displayer"
	"github.com/reanahub/reana-client-go/pkg/tester"

	"github.com/spf13/cobra"
)

const testDesc = `
Test workflow execution, based on a given Gherkin file.

Gherkin files can be specified in the reana specification file (reana.yaml),
or by using the ` + "``-n``" + ` option.

The ` + "``test``" + ` command allows for testing of a workflow execution by
assessing whether it meets properties specified in the selected Gherkin files.

Examples:

  $ reana-client test -w myanalysis -n test_analysis.feature

  $ reana-client test -w myanalysis

  $ reana-client test -w myanalysis -n test1.feature -n test2.feature
`

type testOptions struct {
	token     string
	workflow  string
	testFiles []string
}

// newTestCmd creates a command to test a completed workflow run.
func newTestCmd() *cobra.Command {
	o := &testOptions{}
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test workflow execution.",
		Long:  testDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringArrayVarP(
		&o.testFiles,
		"test-files",
		"n",
		nil,
		"Gherkin file for testing properties of a workflow execution. Overrides files in reana.yaml if provided.",
	)
	f.StringVarP(
		&o.token,
		"access-token",
		"t",
		"",
		"Access token of the current user.",
	)
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w",
		"",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	return cmd
}

func (o *testOptions) run(cmd *cobra.Command) error {
	startedAt := time.Now()
	api, err := client.ApiClient(o.token)
	if err != nil {
		return err
	}
	dataFetcher := tester.NewAPIFetcher(api)
	status, err := dataFetcher.Status(o.workflow)
	if err != nil {
		return fmt.Errorf("could not find workflow %q: %v", o.workflow, err)
	}
	if status.Status != "finished" {
		return fmt.Errorf(
			"%q is %s; it must be finished to run tests",
			o.workflow,
			status.Status,
		)
	}

	testFiles := o.testFiles
	if len(testFiles) == 0 {
		specification, err := dataFetcher.Specification(o.workflow)
		if err != nil {
			return err
		}
		testFiles = specification.TestFiles
		if len(testFiles) == 0 {
			return fmt.Errorf(
				"no test files specified in reana.yaml and no -n option provided",
			)
		}
	}

	passed := 0
	failed := 0
	out := cmd.OutOrStdout()
	for _, testFile := range testFiles {
		fmt.Fprintln(out)
		displayer.DisplayMessage(
			fmt.Sprintf("Testing file %q...", testFile),
			displayer.Info,
			false,
			out,
		)
		_, results, err := tester.ParseAndRun(
			testFile,
			status.Name,
			dataFetcher,
		)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("test file %s not found", testFile)
			}
			var featureFileError *tester.FeatureFileError
			if errors.As(err, &featureFileError) {
				return fmt.Errorf(
					"error parsing feature file %s: %w",
					testFile,
					err,
				)
			}
			return err
		}
		for _, result := range results {
			messageType := displayer.Success
			if result.Status == tester.Failed {
				messageType = displayer.Error
				failed++
			} else {
				// Keep Python-client output compatibility: scenarios skipped by
				// a Gherkin precondition are reported in the passing total.
				passed++
			}
			displayer.DisplayMessage(
				fmt.Sprintf("Scenario %q", result.Scenario),
				messageType,
				true,
				out,
			)
		}
	}

	duration := time.Since(startedAt).Seconds()
	fmt.Fprintf(
		out,
		"\n%d passed, %d failed in %.0fs\n",
		passed,
		failed,
		duration,
	)
	if failed > 0 {
		return config.ErrEmpty
	}
	return nil
}
