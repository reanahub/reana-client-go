/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"
	"time"

	"github.com/spf13/cobra"
)

const listFormatFlagDesc = `Format output according to column titles or column
values. Use <columm_name>=<column_value> format.
E.g. display workflow with failed status and named test_workflow
--format status=failed,name=test_workflow.
`

const listFilterFlagDesc = `Filter workflow that contains certain filtering
criteria. Use --filter
<columm_name>=<column_value> pairs. Available
filters are 'name' and 'status'.
`

const listDesc = `
List all workflows and sessions.

The ` + "``list``" + ` command lists workflows and sessions. By default, the list of
workflows is returned. If you would like to see the list of your open
interactive sessions, you need to pass the ` + "``--sessions``" + ` command-line
option.

Example:

  $ reana-client list --all

  $ reana-client list --sessions

  $ reana-client list --verbose --bytes
`

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workflows and sessions.",
		Long:  listDesc,
		Run: func(cmd *cobra.Command, args []string) {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")
			validation.ValidateAccessToken(token)
			validation.ValidateServerURL(serverURL)
			list(cmd)
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().StringP("workflow", "w", "", "List all runs of the given workflow.")
	cmd.Flags().BoolP("sessions", "s", false, "List all open interactive sessions.")
	cmd.Flags().StringSlice("format", []string{}, listFormatFlagDesc)
	cmd.Flags().Bool("json", false, "Get output in JSON format.")
	cmd.Flags().Bool("all", false, "Show all workflows including deleted ones.")
	cmd.Flags().
		BoolP("verbose", "v", false, `Print out extra information: workflow id, user id, disk usage,
progress, duration.`)
	cmd.Flags().BoolP("human-readable", "r", false, "Show disk size in human readable format.")
	cmd.Flags().String("sort", "CREATED", "Sort the output by specified column.")
	cmd.Flags().StringArray("filter", []string{}, listFilterFlagDesc)
	cmd.Flags().
		Bool("include-duration", false, `Include the duration of the workflows in seconds. In case a workflow is in
progress, its duration as of now will be shown.`)
	cmd.Flags().Bool("include-progress", false, "Include progress information of the workflows.")
	cmd.Flags().Bool("include-workspace-size", false, "Include size information of the workspace.")
	cmd.Flags().Bool("show-deleted-runs", false, "Include deleted workflows in the output.")
	cmd.Flags().Int64("page", 1, "Results page number (to be used with --size).")
	cmd.Flags().Int64("size", 0, "Size of results per page (to be used with --page).")

	return cmd
}

func list(cmd *cobra.Command) {
	token, _ := cmd.Flags().GetString("access-token")
	if token == "" {
		token = os.Getenv("REANA_ACCESS_TOKEN")
	}
	workflow, _ := cmd.Flags().GetString("workflow")
	listSessions, _ := cmd.Flags().GetBool("sessions")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")
	verbose, _ := cmd.Flags().GetBool("verbose")
	humanReadable, _ := cmd.Flags().GetBool("human-readable")
	filters, _ := cmd.Flags().GetStringArray("filter")
	includeDuration, _ := cmd.Flags().GetBool("include-duration")
	includeProgress, _ := cmd.Flags().GetBool("include-progress")
	includeWorkspaceSize, _ := cmd.Flags().GetBool("include-workspace-size")
	showDeletedRuns, _ := cmd.Flags().GetBool("show-deleted-runs")
	page, _ := cmd.Flags().GetInt64("page")
	size, _ := cmd.Flags().GetInt64("size")

	var runType string
	if listSessions {
		runType = "interactive"
	} else {
		runType = "batch"
	}

	statusFilters := utils.GetRunStatuses(showDeletedRuns || showAll)
	var searchFilter string
	if len(filters) > 0 {
		filterNames := []string{"name", "status"}
		statusFilters, searchFilter = utils.ParseListFilters(filters, filterNames)
	}

	listParams := operations.NewGetWorkflowsParams()
	listParams.SetAccessToken(&token)
	listParams.SetType(runType)
	listParams.SetVerbose(&verbose)
	listParams.SetPage(&page)
	listParams.SetWorkflowIDOrName(&workflow)
	listParams.SetStatus(statusFilters)
	listParams.SetSearch(&searchFilter)
	listParams.SetIncludeProgress(&includeProgress)
	listParams.SetIncludeWorkspaceSize(&includeWorkspaceSize)
	if cmd.Flags().Changed("size") {
		listParams.SetSize(&size)
	}

	listResp, err := client.ApiClient().Operations.GetWorkflows(listParams)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	header := buildListHeader(
		runType,
		verbose,
		includeWorkspaceSize,
		includeProgress,
		includeDuration,
	)
	displayListPayload(listResp.Payload, header, jsonOutput, humanReadable)
}

func displayListPayload(
	p *operations.GetWorkflowsOKBody,
	header []string,
	jsonOutput, humanReadable bool,
) {
	var data [][]any
	for _, workflow := range p.Items {
		name, runNumber := utils.GetWorkflowNameAndRunNumber(workflow.Name)

		var row []any
		for _, col := range header {
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
					value = workflow.Size.Raw
				}
			case "progress":
				progress := workflow.Progress
				finishedInfo := getOptionalIntField(progress.Finished.Total)
				totalInfo := getOptionalIntField(progress.Total.Total)
				value = finishedInfo + "/" + totalInfo
			case "duration":
				value = getWorkflowDuration(workflow)
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
				// TODO
			case "session_uri":
				// TODO
			case "session_status":
				// TODO
			}

			row = append(row, value)
		}
		data = append(data, row)
	}

	if jsonOutput {
		// TODO Fix json output
		utils.DisplayJsonOutput(data)
	} else {
		utils.DisplayTable(header, data)
	}
}

func getWorkflowDuration(workflow *operations.GetWorkflowsOKBodyItemsItems0) string {
	runStartedAt := workflow.Progress.RunStartedAt
	runFinishedAt := workflow.Progress.RunFinishedAt
	if runStartedAt == nil {
		return "-"
	}
	startTime := utils.FromIsoToTimestamp(*runStartedAt)
	var endTime time.Time
	if runFinishedAt != nil {
		endTime = utils.FromIsoToTimestamp(*runFinishedAt)
	} else {
		endTime = time.Now()
	}
	duration := endTime.Sub(startTime).Round(time.Second).Seconds()
	return fmt.Sprintf("%g", duration)
}

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

	if verbose {
		headers[runType] = append(headers[runType], "id", "user")
	}
	if verbose || includeWorkspaceSize {
		headers[runType] = append(headers[runType], "size")
	}
	if verbose || includeProgress {
		headers[runType] = append(headers[runType], "progress")
	}
	if verbose || includeDuration {
		headers[runType] = append(headers[runType], "duration")
	}

	return headers[runType]
}

func getOptionalStringField(value *string) string {
	if value == nil {
		return "-"
	}
	return *value
}

func getOptionalIntField(value int64) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}
