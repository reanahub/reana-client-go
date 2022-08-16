/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"

	"github.com/spf13/cobra"
)

const statusDesc = `
Get status of a workflow.

The ` + "``status``" + ` command allow to retrieve status of a workflow. The status
can be created, queued, running, failed, etc. You can increase verbosity or
filter retrieved information by passing appropriate command-line options.

Examples:

  $ reana-client status -w myanalysis.42

  $ reana-client status -w myanalysis.42 -v --json
`

const statusFormatFlagDesc = `Format output by displaying only certain columns.
E.g. --format name,status.`

type statusOptions struct {
	token           string
	workflow        string
	formatFilters   []string
	jsonOutput      bool
	verbose         bool
	includeDuration bool
}

// newStatusCmd creates a command to get status of a workflow.
func newStatusCmd() *cobra.Command {
	o := &statusOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get status of a workflow.",
		Long:  statusDesc,
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
	f.StringSliceVar(&o.formatFilters, "format", []string{}, statusFormatFlagDesc)
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

func (o *statusOptions) run(cmd *cobra.Command) error {
	statusParams := operations.NewGetWorkflowStatusParams()
	statusParams.SetAccessToken(&o.token)
	statusParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	statusResp, err := api.Operations.GetWorkflowStatus(statusParams)
	if err != nil {
		return err
	}
	payload := statusResp.Payload

	header := buildStatusHeader(
		o.verbose,
		o.includeDuration,
		payload.Progress,
		payload.Status,
	)
	parsedFormatFilters := utils.ParseFormatParameters(o.formatFilters, false)
	err = displayStatusPayload(
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

// displayStatusPayload displays the status payload, according to the given header, filters and output format.
func displayStatusPayload(
	cmd *cobra.Command,
	p *operations.GetWorkflowStatusOKBody,
	header []string,
	filters []utils.FormatFilter,
	jsonOutput bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := buildStatusSeries(col)
		name, runNumber := utils.GetWorkflowNameAndRunNumber(p.Name)
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
			value = getStatusProgress(p.Progress)
		case "started":
			value = p.Progress.RunStartedAt
		case "ended":
			value = p.Progress.RunFinishedAt
		case "id":
			value = p.ID
		case "user":
			value = p.User
		case "command":
			value = getStatusCommand(p.Progress)
		case "duration":
			var err error
			value, err = utils.GetWorkflowDuration(
				&p.Progress.RunStartedAt,
				&p.Progress.RunFinishedAt,
			)
			if err != nil {
				return err
			}
		}

		colSeries.Append(value)
		df = df.CBind(dataframe.New(colSeries))
	}

	df = utils.FormatDataFrame(df, filters)

	if jsonOutput {
		err := utils.DisplayJsonOutput(df.Maps(), cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		data := utils.DataFrameToStringData(df)
		utils.DisplayTable(df.Names(), data, cmd.OutOrStdout())
	}

	return nil
}

// buildStatusHeader builds the header of the status table, according to whether to include
// verbose information and additional headers.
func buildStatusHeader(
	verbose bool,
	includeDuration bool,
	progress *operations.GetWorkflowStatusOKBodyProgress,
	status string,
) []string {
	headers := []string{"name", "run_number", "created"}

	includeProgress := progress.Total != nil
	hasRunStarted := slices.Contains([]string{"running", "finished", "failed", "stopped"}, status)
	includeStarted := progress.RunStartedAt != ""
	includeEnded := progress.RunFinishedAt != ""
	includeCommand := progress.CurrentCommand != "" || progress.CurrentStepName != ""

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

// getStatusProgress formats the progress of the workflow as finished/total.
func getStatusProgress(progress *operations.GetWorkflowStatusOKBodyProgress) any {
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

// getStatusCommand gets the current command of the workflow.
// If the command isn't available, it returns the current step name.
func getStatusCommand(progress *operations.GetWorkflowStatusOKBodyProgress) string {
	currentCmd := progress.CurrentCommand
	if currentCmd == "" {
		return progress.CurrentStepName
	}
	if strings.HasPrefix(currentCmd, "bash -c \"cd ") {
		commaIdx := strings.Index(currentCmd, ";")
		currentCmd = currentCmd[commaIdx+2 : len(currentCmd)-2]
	}
	return currentCmd
}

// buildStatusSeries returns a Series of the right type, according to the column name.
func buildStatusSeries(col string) series.Series {
	if col == "duration" {
		return series.New([]int{}, series.Int, col)
	}
	return series.New([]string{}, series.String, col)
}
