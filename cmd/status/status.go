/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package status provides the command to get status of a workflow.
package status

import (
	"fmt"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/formatter"
	"reanahub/reana-client-go/pkg/workflows"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"

	"github.com/spf13/cobra"
)

const description = `
Get status of a workflow.

The ` + "``status``" + ` command allow to retrieve status of a workflow. The status
can be created, queued, running, failed, etc. You can increase verbosity or
filter retrieved information by passing appropriate command-line options.

Examples:

  $ reana-client status -w myanalysis.42

  $ reana-client status -w myanalysis.42 -v --json
`

const formatFlagDesc = `Format output by displaying only certain columns.
E.g. --format name,status.`

type options struct {
	token           string
	workflow        string
	formatFilters   []string
	jsonOutput      bool
	verbose         bool
	includeDuration bool
}

// NewCmd creates a command to get status of a workflow.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get status of a workflow.",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w", "",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	f.StringSliceVar(&o.formatFilters, "format", []string{}, formatFlagDesc)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	f.BoolVarP(&o.verbose, "verbose", "v", false, "Set status information verbosity.")
	f.BoolVar(
		&o.includeDuration,
		"include-duration",
		false,
		`Include the duration of the workflows in seconds.
In case a workflow is in progress, its duration as of now will be shown.`,
	)

	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
	payload, err := workflows.GetStatus(o.token, o.workflow)
	if err != nil {
		return err
	}

	header := buildHeader(
		o.verbose,
		o.includeDuration,
		payload.Progress,
		payload.Status,
	)
	parsedFormatFilters := formatter.ParseFormatParameters(o.formatFilters, false)
	err = displayPayload(
		cmd,
		payload,
		header,
		parsedFormatFilters,
		o.jsonOutput,
	)
	if err != nil {
		return err
	}

	return nil
}

// displayPayload displays the status payload, according to the given header, filters and output format.
func displayPayload(
	cmd *cobra.Command,
	p *operations.GetWorkflowStatusOKBody,
	header []string,
	filters []formatter.FormatFilter,
	jsonOutput bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := buildSeries(col)
		name, runNumber := workflows.GetNameAndRunNumber(p.Name)
		var value any

		switch col {
		case "name":
			value = name
		case "run_number":
			value = runNumber
		case "created":
			value = p.Created
		case "status":
			value = p.Status
		case "progress":
			value = formatProgress(p.Progress)
		case "started":
			value = *p.Progress.RunStartedAt
		case "ended":
			value = *p.Progress.RunFinishedAt
		case "id":
			value = p.ID
		case "user":
			value = p.User
		case "command":
			value = getCurrentCommand(p.Progress)
		case "duration":
			var err error
			value, err = workflows.GetDuration(
				p.Progress.RunStartedAt,
				p.Progress.RunFinishedAt,
			)
			if err != nil {
				return err
			}
		}

		colSeries.Append(value)
		df = df.CBind(dataframe.New(colSeries))
	}

	df, err := formatter.FormatDataFrame(df, filters)
	if err != nil {
		return err
	}

	if jsonOutput {
		err := displayer.DisplayJsonOutput(df.Maps(), cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		data := formatter.DataFrameToStringData(df)
		displayer.DisplayTable(df.Names(), data, cmd.OutOrStdout())
	}

	return nil
}

// buildHeader builds the header of the status table, according to whether to include
// verbose information and additional headers.
func buildHeader(
	verbose bool,
	includeDuration bool,
	progress *operations.GetWorkflowStatusOKBodyProgress,
	status string,
) []string {
	headers := []string{"name", "run_number", "created"}

	includeProgress := progress.Total != nil
	hasRunStarted := slices.Contains([]string{"running", "finished", "failed", "stopped"}, status)
	includeStarted := progress.RunStartedAt != nil
	includeEnded := progress.RunFinishedAt != nil
	includeCommand := progress.CurrentCommand != nil || progress.CurrentStepName != nil

	if hasRunStarted && includeStarted {
		headers = append(headers, "started")
		if includeEnded {
			headers = append(headers, "ended")
		}
	}
	headers = append(headers, "status")
	if includeProgress {
		headers = append(headers, "progress")
	}
	if verbose {
		headers = append(headers, "id", "user")
		if includeCommand {
			headers = append(headers, "command")
		}
	}
	if verbose || includeDuration {
		headers = append(headers, "duration")
	}
	return headers
}

// formatProgress formats the progress of the workflow as finished/total.
func formatProgress(progress *operations.GetWorkflowStatusOKBodyProgress) any {
	var totalJobs, finishedJobs int64
	if progress.Total != nil {
		totalJobs = progress.Total.Total
	}
	if progress.Finished != nil {
		finishedJobs = progress.Finished.Total
	}
	if totalJobs > 0 {
		return fmt.Sprintf("%d/%d", finishedJobs, totalJobs)
	}
	return "-/-"
}

// getCurrentCommand gets the current command of the workflow.
// If the command isn't available, it returns the current step name.
func getCurrentCommand(progress *operations.GetWorkflowStatusOKBodyProgress) string {
	if progress.CurrentCommand == nil {
		return *progress.CurrentStepName
	}
	currentCmd := *progress.CurrentCommand
	if strings.HasPrefix(currentCmd, "bash -c \"cd ") {
		commaIdx := strings.Index(currentCmd, ";")
		currentCmd = currentCmd[commaIdx+2 : len(currentCmd)-2]
	}
	return currentCmd
}

// buildSeries returns a Series of the right type, according to the column name.
func buildSeries(col string) series.Series {
	if col == "duration" {
		return series.New([]int{}, series.Int, col)
	}
	return series.New([]string{}, series.String, col)
}
