/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/slices"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")

			if err := validation.ValidateAccessToken(token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(serverURL); err != nil {
				return err
			}
			if err := list(cmd, token, serverURL); err != nil {
				return err
			}
			return nil
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
	cmd.Flags().StringSlice("filter", []string{}, listFilterFlagDesc)
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

func list(cmd *cobra.Command, token string, serverURL string) error {
	workflow, _ := cmd.Flags().GetString("workflow")
	listSessions, _ := cmd.Flags().GetBool("sessions")
	formatFilters, _ := cmd.Flags().GetStringSlice("format")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")
	verbose, _ := cmd.Flags().GetBool("verbose")
	humanReadable, _ := cmd.Flags().GetBool("human-readable")
	sortColumn, _ := cmd.Flags().GetString("sort")
	filters, _ := cmd.Flags().GetStringSlice("filter")
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
		var err error
		statusFilters, searchFilter, err = utils.ParseFilterParameters(filters, filterNames)
		if err != nil {
			return err
		}
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
		verbose,
		includeWorkspaceSize,
		includeProgress,
		includeDuration,
	)
	parsedFormatFilters := utils.ParseFormatParameters(formatFilters)
	err = displayListPayload(
		cmd,
		listResp.Payload,
		header,
		parsedFormatFilters,
		serverURL,
		token,
		sortColumn,
		jsonOutput,
		humanReadable,
	)
	if err != nil {
		return err
	}

	return nil
}

func displayListPayload(
	cmd *cobra.Command,
	p *operations.GetWorkflowsOKBody,
	header []string,
	formatFilters map[string]string,
	serverURL, token, sortColumn string,
	jsonOutput, humanReadable bool,
) error {
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
				var err error
				value, err = getWorkflowDuration(workflow)
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
				if workflow.SessionURI == "" {
					value = "-"
				} else {
					value = utils.FormatSessionURI(serverURL, workflow.SessionURI, token)
				}
			case "session_status":
				value = getOptionalStringField(&workflow.SessionStatus)
			}

			row = append(row, value)
		}
		data = append(data, row)
	}

	err := sortListData(data, header, sortColumn)
	if err != nil {
		cmd.PrintErrf("Warning: sort operation was aborted, \"%v\"\n", err)
	}
	utils.FormatData(&data, &header, formatFilters)

	if jsonOutput {
		jsonData := make([]map[string]any, len(data))
		for i, row := range data {
			jsonData[i] = map[string]any{}
			for j, col := range header {
				jsonData[i][col] = row[j]
			}
		}

		err := utils.DisplayJsonOutput(jsonData, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		utils.DisplayTable(header, data, cmd.OutOrStdout())
	}
	return nil
}

func getWorkflowDuration(workflow *operations.GetWorkflowsOKBodyItemsItems0) (any, error) {
	runStartedAt := workflow.Progress.RunStartedAt
	runFinishedAt := workflow.Progress.RunFinishedAt
	if runStartedAt == nil {
		return "-", nil
	}

	startTime, err := utils.FromIsoToTimestamp(*runStartedAt)
	if err != nil {
		return nil, err
	}

	var endTime time.Time
	if runFinishedAt != nil {
		endTime, err = utils.FromIsoToTimestamp(*runFinishedAt)
		if err != nil {
			return nil, err
		}
	} else {
		endTime = time.Now()
	}
	return endTime.Sub(startTime).Round(time.Second).Seconds(), nil
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

func getOptionalStringField(value *string) string {
	if value == nil || *value == "" {
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

func sortListData(data [][]any, header []string, sortColumn string) error {
	sortColumn = strings.ToLower(sortColumn)
	if !slices.Contains(header, sortColumn) {
		return fmt.Errorf("column '%s' does not exist", sortColumn)
	}

	sortColumnId := slices.Index(header, sortColumn)
	ok := true
	sort.SliceStable(data, func(i, j int) bool {
		value1 := data[i][sortColumnId]
		value2 := data[j][sortColumnId]

		// Make sure missing values are at the bottom of the list
		if value1 == "-" {
			return false
		}
		if value2 == "-" {
			return true
		}

		switch value1.(type) {
		case int64:
			return value1.(int64) > value2.(int64)
		case float64:
			return value1.(float64) > value2.(float64)
		case string:
			return value1.(string) > value2.(string)
		default:
			ok = false
			return false
		}
	})

	if !ok {
		return errors.New("unexpected value type received")
	}
	return nil
}
