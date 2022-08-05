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

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const listFormatFlagDesc = `Format output according to column titles or column
values. Use <columm_name>=<column_value> format.
E.g. display workflow with failed status and named test_workflow
--format status=failed,name=test_workflow.`

const listFilterFlagDesc = `Filter workflow that contains certain filtering
criteria. Use --filter
<columm_name>=<column_value> pairs. Available
filters are 'name' and 'status'.`

const listDesc = `
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

type listOptions struct {
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

// newListCmd creates a new command for listing workflows and sessions.
func newListCmd() *cobra.Command {
	o := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workflows and sessions.",
		Long:  listDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(&o.workflow, "workflow", "w", "", "List all runs of the given workflow.")
	f.BoolVarP(&o.listSessions, "sessions", "s", false, "List all open interactive sessions.")
	f.StringSliceVar(&o.formatFilters, "format", []string{}, listFormatFlagDesc)
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
		"r",
		false,
		"Show disk size in human readable format.",
	)
	f.StringVar(&o.sortColumn, "sort", "CREATED", "Sort the output by specified column.")
	f.StringSliceVar(&o.filters, "filter", []string{}, listFilterFlagDesc)
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

	err := f.SetAnnotation("workflow", "properties", []string{"optional"})
	if err != nil {
		log.Debugf("Failed to set workflow annotation: %s", err.Error())
	}
	return cmd
}

func (o *listOptions) run(cmd *cobra.Command) error {
	var runType string
	if o.listSessions {
		runType = "interactive"
	} else {
		runType = "batch"
	}

	statusFilters, searchFilter, err := parseListFilters(o.filters, o.showDeletedRuns, o.showAll)
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
	listParams.SetIncludeProgress(&o.includeProgress)
	listParams.SetIncludeWorkspaceSize(&o.includeWorkspaceSize)
	if cmd.Flags().Changed("size") {
		listParams.SetSize(&o.size)
	}

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	listResp, err := api.Operations.GetWorkflows(listParams)
	if err != nil {
		return err
	}

	header := buildListHeader(
		runType,
		o.verbose,
		o.includeWorkspaceSize,
		o.includeProgress,
		o.includeDuration,
	)
	parsedFormatFilters := utils.ParseFormatParameters(o.formatFilters, true)
	err = displayListPayload(
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

// displayListPayload displays the list payload, according to the given header, filters and output format.
func displayListPayload(
	cmd *cobra.Command,
	p *operations.GetWorkflowsOKBody,
	header []string,
	formatFilters []utils.FormatFilter,
	serverURL, token, sortColumn string,
	jsonOutput, humanReadable bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := buildListSeries(col, humanReadable)
		for _, workflow := range p.Items {
			name, runNumber := utils.GetWorkflowNameAndRunNumber(workflow.Name)
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
				value, err = utils.GetWorkflowDuration(
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
					value = utils.FormatSessionURI(serverURL, workflow.SessionURI, token)
				}
			case "session_status":
				value = getOptionalStringField(&workflow.SessionStatus)
			}

			colSeries.Append(value)
		}

		df = df.CBind(dataframe.New(colSeries))
	}

	df, err := utils.SortDataFrame(df, sortColumn, true)
	if err != nil {
		cmd.PrintErrf("Warning: sort operation was aborted, %s\n", err)
	}
	df = utils.FormatDataFrame(df, formatFilters)

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

// buildListHeader builds the header of the list table, according to the given runType and whether to include
// verbose information, workspace size, progress and duration.
func buildListHeader(
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

// parseListFilters takes the filter input and returns status filters as a slice and the remaining filters
// as a JSON string, according to whether it should show deleted status.
func parseListFilters(
	filterInput []string,
	showDeletedRuns, showAll bool,
) ([]string, string, error) {
	filterNames := []string{"name", "status"}
	filters, err := utils.NewFilters(nil, filterNames, filterInput)
	if err != nil {
		return nil, "", err
	}

	statusFilters := utils.GetRunStatuses(showDeletedRuns || showAll)
	err = filters.ValidateValues("status", utils.GetRunStatuses(true))
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

	searchFilter, err := filters.GetJson([]string{"name"}) // All the filters except for status
	if err != nil {
		return nil, "", err
	}

	return statusFilters, searchFilter, nil
}

// buildListSeries returns a Series of the right type, according to the column name.
func buildListSeries(col string, humanReadable bool) series.Series {
	if col == "duration" || (col == "size" && !humanReadable) {
		return series.New([]int{}, series.Int, col)
	}
	return series.New([]string{}, series.String, col)
}

// getOptionalStringField returns the given string field, if it is not nil or empty, otherwise "-".
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
