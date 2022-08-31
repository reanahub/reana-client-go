/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package list provides the command for listing workflows and sessions.
package list

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/datautils"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/filterer"
	"reanahub/reana-client-go/pkg/formatter"
	"reanahub/reana-client-go/pkg/workflows"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const description = `
List all workflows and sessions.

The ` + "``list``" + ` command lists workflows and sessions. By default, the list of
workflows is returned. If you would like to see the list of your open
interactive sessions, you need to pass the ` + "``--sessions``" + ` command-line
option.

Examples:

  $ reana-client list --all

  $ reana-client list --sessions

  $ reana-client list --verbose --bytes
`

const formatFlagDesc = `Format output according to column titles or column
values. Use <columm_name>=<column_value> format.
E.g. display workflow with failed status and named test_workflow
--format status=failed,name=test_workflow.`

const filterFlagDesc = `Filter workflow that contains certain filtering
criteria. Use --filter
<columm_name>=<column_value> pairs. Available
filters are 'name' and 'status'.`

type options struct {
	token                string
	serverURL            string
	workflow             string
	listSessions         bool
	formatFilters        []string
	jsonOutput           bool
	showAll              bool
	verbose              bool
	humanReadable        bool
	sortColumn           string
	filters              []string
	includeDuration      bool
	includeProgress      bool
	includeWorkspaceSize bool
	showDeletedRuns      bool
	page                 int64
	size                 int64
}

// NewCmd creates a new command for listing workflows and sessions.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workflows and sessions.",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(&o.workflow, "workflow", "w", "", "List all runs of the given workflow.")
	f.BoolVarP(&o.listSessions, "sessions", "s", false, "List all open interactive sessions.")
	f.StringSliceVar(&o.formatFilters, "format", []string{}, formatFlagDesc)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	f.BoolVar(&o.showAll, "all", false, "Show all workflows including deleted ones.")
	f.BoolVarP(
		&o.verbose,
		"verbose",
		"v",
		false,
		`Print out extra information: workflow id, user id, disk usage,
progress, duration.`,
	)
	f.BoolVarP(
		&o.humanReadable,
		"human-readable",
		"h",
		false,
		"Show disk size in human readable format.",
	)
	f.StringVar(&o.sortColumn, "sort", "CREATED", "Sort the output by specified column.")
	f.StringSliceVar(&o.filters, "filter", []string{}, filterFlagDesc)
	f.BoolVar(
		&o.includeDuration,
		"include-duration",
		false,
		`Include the duration of the workflows in seconds.
In case a workflow is in progress, its duration as of now will be shown.`,
	)
	f.BoolVar(
		&o.includeProgress,
		"include-progress",
		false,
		"Include progress information of the workflows.",
	)
	f.BoolVar(
		&o.includeWorkspaceSize,
		"include-workspace-size",
		false,
		"Include size information of the workspace.",
	)
	f.BoolVar(
		&o.showDeletedRuns,
		"show-deleted-runs",
		false,
		"Include deleted workflows in the output.",
	)
	f.Int64Var(&o.page, "page", 1, "Results page number (to be used with --size).")
	f.Int64Var(&o.size, "size", 0, "Number of results per page (to be used with --page).")
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "", false, "Help for du")

	err := f.SetAnnotation("workflow", "properties", []string{"optional"})
	if err != nil {
		log.Debugf("Failed to set workflow annotation: %s", err.Error())
	}
	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
	var runType string
	if o.listSessions {
		runType = "interactive"
	} else {
		runType = "batch"
	}

	statusFilters, searchFilter, err := parseFilters(o.filters, o.showDeletedRuns, o.showAll)
	if err != nil {
		return err
	}

	listParams := operations.NewGetWorkflowsParams()
	listParams.SetAccessToken(&o.token)
	listParams.SetType(runType)
	listParams.SetVerbose(&o.verbose)
	listParams.SetPage(&o.page)
	listParams.SetWorkflowIDOrName(&o.workflow)
	listParams.SetStatus(statusFilters)
	listParams.SetSearch(&searchFilter)
	if cmd.Flags().Changed("size") {
		listParams.SetSize(&o.size)
	}
	// Don't set these to false because they override the server's verbose flag
	if cmd.Flags().Changed("include-progress") {
		listParams.SetIncludeProgress(&o.includeProgress)
	}
	if cmd.Flags().Changed("include-workspace-size") {
		listParams.SetIncludeWorkspaceSize(&o.includeWorkspaceSize)
	}

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	listResp, err := api.Operations.GetWorkflows(listParams)
	if err != nil {
		return err
	}

	header := buildHeader(
		runType,
		o.verbose,
		o.includeWorkspaceSize,
		o.includeProgress,
		o.includeDuration,
	)
	parsedFormatFilters := formatter.ParseFormatParameters(o.formatFilters, true)
	err = displayPayload(
		cmd,
		listResp.Payload,
		header,
		parsedFormatFilters,
		o.serverURL,
		o.token,
		o.sortColumn,
		o.jsonOutput,
		o.humanReadable,
	)
	if err != nil {
		return err
	}

	return nil
}

// displayPayload displays the list payload, according to the given header, filters and output format.
func displayPayload(
	cmd *cobra.Command,
	p *operations.GetWorkflowsOKBody,
	header []string,
	formatFilters []formatter.FormatFilter,
	serverURL, token, sortColumn string,
	jsonOutput, humanReadable bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := buildSeries(col, humanReadable)
		for _, workflow := range p.Items {
			name, runNumber := workflows.GetNameAndRunNumber(workflow.Name)
			var value any

			switch col {
			case "id":
				value = workflow.ID
			case "user":
				value = workflow.User
			case "size":
				if humanReadable {
					value = workflow.Size.HumanReadable
				} else {
					value = int(workflow.Size.Raw)
				}
			case "progress":
				progress := workflow.Progress
				finishedInfo := getProgressField(progress.Finished.Total)
				totalInfo := getProgressField(progress.Total.Total)
				value = finishedInfo + "/" + totalInfo
			case "duration":
				var err error
				value, err = workflows.GetDuration(
					workflow.Progress.RunStartedAt,
					workflow.Progress.RunFinishedAt,
				)
				if err != nil {
					return err
				}
			case "name":
				value = name
			case "run_number":
				value = runNumber
			case "created":
				value = workflow.Created
			case "started":
				value = getOptionalStringField(workflow.Progress.RunStartedAt)
			case "ended":
				value = getOptionalStringField(workflow.Progress.RunFinishedAt)
			case "status":
				value = workflow.Status
			case "session_type":
				value = getOptionalStringField(&workflow.SessionType)
			case "session_uri":
				if workflow.SessionURI != "" {
					value = formatter.FormatSessionURI(serverURL, workflow.SessionURI, token)
				}
			case "session_status":
				value = getOptionalStringField(&workflow.SessionStatus)
			}

			colSeries.Append(value)
		}

		df = df.CBind(dataframe.New(colSeries))
	}

	df, err := formatter.SortDataFrame(df, sortColumn, true)
	if err != nil {
		cmd.PrintErrf("Warning: sort operation was aborted, %s\n", err)
	}
	df, err = formatter.FormatDataFrame(df, formatFilters)
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

// buildHeader builds the header of the list table, according to the given runType and whether to include
// verbose information, workspace size, progress and duration.
func buildHeader(
	runType string,
	verbose, includeWorkspaceSize, includeProgress, includeDuration bool,
) []string {
	headers := map[string][]string{
		"batch": {"name", "run_number", "created", "started", "ended", "status"},
		"interactive": {
			"name",
			"run_number",
			"created",
			"session_type",
			"session_uri",
			"session_status",
		},
	}

	header := headers[runType]
	if verbose {
		header = append(header, "id", "user")
	}
	if verbose || includeWorkspaceSize {
		header = append(header, "size")
	}
	if verbose || includeProgress {
		header = append(header, "progress")
	}
	if verbose || includeDuration {
		header = append(header, "duration")
	}

	return header
}

// parseFilters takes the filter input and returns status filters as a slice and the remaining filters
// as a JSON string, according to whether it should show deleted status.
func parseFilters(
	filterInput []string,
	showDeletedRuns, showAll bool,
) ([]string, string, error) {
	filters, err := filterer.NewFilters(nil, config.ListMultiFilters, filterInput)
	if err != nil {
		return nil, "", err
	}

	statusFilters := config.GetRunStatuses(showDeletedRuns || showAll)
	err = filters.ValidateValues("status", config.GetRunStatuses(true))
	if err != nil {
		return nil, "", err
	}
	userStatusFilters, err := filters.GetMulti("status")
	if err != nil {
		return nil, "", err
	}
	if len(userStatusFilters) > 0 {
		statusFilters = userStatusFilters
	}

	jsonFilters := datautils.RemoveFromSlice(config.ListMultiFilters, "status")
	searchFilter, err := filters.GetJson(jsonFilters)
	if err != nil {
		return nil, "", err
	}

	return statusFilters, searchFilter, nil
}

// buildSeries returns a Series of the right type, according to the column name.
func buildSeries(col string, humanReadable bool) series.Series {
	if col == "duration" || (col == "size" && !humanReadable) {
		return series.New([]int{}, series.Int, col)
	}
	return series.New([]string{}, series.String, col)
}

// getOptionalStringField returns the given string field, if it is not nil or empty, otherwise nil.
func getOptionalStringField(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

// getProgressField returns the given int value converted to string, if it is not 0, otherwise "-".
func getProgressField(value int64) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}
