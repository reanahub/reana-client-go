/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var listFormatFlagDescription = `Format output according to column titles or column
values. Use <columm_name>=<column_value> format.
E.g. display workflow with failed status and named test_workflow
--format status=failed,name=test_workflow.
`

var listFilterFlagDescription = `Filter workflow that contains certain filtering
criteria. Use --filter
<columm_name>=<column_value> pairs. Available
filters are 'name' and 'status'.
`

// Available run statuses
var runStatuses = []string{"created", "running", "finished", "failed", "deleted", "stopped", "queued", "pending"}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workflows and sessions.",
		Long: `List all workflows and sessions.

	The ` + "``list``" + ` command lists workflows and sessions. By default, the list of
	workflows is returned. If you would like to see the list of your open
	interactive sessions, you need to pass the ` + "``--sessions``" + ` command-line
	option.

	Example:

	  $ reana-client list --all

	  $ reana-client list --sessions

	  $ reana-client list --verbose --bytes

	  `,
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
	cmd.Flags().StringP("sessions", "s", "", "List all open interactive sessions.")
	cmd.Flags().String("format", "", listFormatFlagDescription)
	cmd.Flags().BoolP("json", "", false, "Get output in JSON format.")
	cmd.Flags().StringArray("filter", []string{}, listFilterFlagDescription)

	return cmd
}

func list(cmd *cobra.Command) {
	token, _ := cmd.Flags().GetString("access-token")
	if token == "" {
		token = os.Getenv("REANA_ACCESS_TOKEN")
	}
	workflow, _ := cmd.Flags().GetString("workflow")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	filter, _ := cmd.Flags().GetStringArray("filter")

	statusFilters, searchFilter := parseListFilters(filter)

	listParams := operations.NewGetWorkflowsParams()
	listParams.SetAccessToken(&token)
	listParams.SetWorkflowIDOrName(&workflow)
	listParams.SetStatus(statusFilters)
	listParams.SetSearch(&searchFilter)

	listResp, err := client.ApiClient.Operations.GetWorkflows(listParams)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	if jsonOutput {
		utils.DisplayJsonOutput(listResp.Payload)
	} else {
		displayListPayload(listResp.Payload)
	}
}

func displayListPayload(p *operations.GetWorkflowsOKBody) {
	fmt.Printf(
		"%-38s %-12s %-21s %-21s %-21s %-8s\n",
		"NAME",
		"RUN_NUMBER",
		"CREATED",
		"STARTED",
		"ENDED",
		"STATUS",
	)
	for _, workflow := range p.Items {
		workflowNameAndRunNumber := strings.SplitN(workflow.Name, ".", 2)
		fmt.Printf(
			"%-38s %-12s %-21s %-21s %-21s %-8s\n",
			workflowNameAndRunNumber[0],
			workflowNameAndRunNumber[1],
			workflow.Created,
			displayOptionalField(workflow.Progress.RunStartedAt),
			displayOptionalField(workflow.Progress.RunFinishedAt),
			workflow.Status,
		)
	}
}

func displayOptionalField(value *string) string {
	if value == nil {
		return "-"
	}
	return *value
}

func parseListFilters(filter []string) ([]string, string) {
	filterNames := []string{"name", "status"}
	searchFilters := make(map[string][]string)
	statusFilters := []string{}

	for _, value := range filter {
		if !strings.Contains(value, "=") {
			fmt.Println("Error: Wrong input format. Please use --filter filter_name=filter_value")
			os.Exit(1)
		}

		filterNameAndValue := strings.SplitN(value, "=", 2)
		filterName := strings.ToLower(filterNameAndValue[0])
		filterValue := filterNameAndValue[1]

		if !slices.Contains(filterNames, filterName) {
			fmt.Printf("Error: Filter %s is not valid", filterName)
			os.Exit(1)
		}

		if filterName == "status" && !slices.Contains(runStatuses, filterValue) {
			fmt.Printf("Error: Input status value %s is not valid. ", filterValue)
			os.Exit(1)
		}

		if filterName == "status" {
			statusFilters = append(statusFilters, filterValue)
		}

		if filterName == "name" {
			searchFilters[filterName] = append(searchFilters[filterName], filterValue)
		}
	}

	searchFiltersString := ""
	if len(searchFilters) > 0 {
		searchFiltersByteArray, err := json.Marshal(searchFilters)
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
		searchFiltersString = string(searchFiltersByteArray)
	}

	return statusFilters, searchFiltersString
}
