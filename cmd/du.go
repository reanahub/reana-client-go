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

	"github.com/spf13/cobra"
)

const duDesc = `
Get workspace disk usage.

The ` + "``du``" + ` command allows to chech the disk usage of given workspace.

Examples:

  $ reana-client du -w myanalysis.42 -s

  $ reana-client du -w myanalysis.42 -s --human-readable

  $ reana-client du -w myanalysis.42 --filter name=data/
`

const duFilterFlagDesc = `Filter results to show only files that match certain filtering
criteria such as file name or size.
Use --filter <columm_name>=<column_value> pairs.
Available filters are 'name' and 'size'.`

func newDuCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "du",
		Short: "Get workspace disk usage.",
		Long:  duDesc,
		Run: func(cmd *cobra.Command, args []string) {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")
			workflow, _ := cmd.Flags().GetString("workflow")
			if workflow == "" {
				workflow = os.Getenv("REANA_WORKON")
			}

			validation.ValidateAccessToken(token)
			validation.ValidateServerURL(serverURL)
			validation.ValidateWorkflow(workflow)
			du(cmd)
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().
		StringP("workflow", "w", "", "Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.")
	cmd.Flags().BoolP("summarize", "s", false, "Display total.")
	cmd.Flags().BoolP("human-readable", "r", false, "Show disk size in human readable format.")
	cmd.Flags().StringArray("filter", []string{}, duFilterFlagDesc)

	return cmd
}

func du(cmd *cobra.Command) {
	token, _ := cmd.Flags().GetString("access-token")
	if token == "" {
		token = os.Getenv("REANA_ACCESS_TOKEN")
	}
	summarize, _ := cmd.Flags().GetBool("summarize")
	humanReadable, _ := cmd.Flags().GetBool("human-readable")
	workflow, _ := cmd.Flags().GetString("workflow")
	if workflow == "" {
		workflow = os.Getenv("REANA_WORKON")
	}
	filter, _ := cmd.Flags().GetStringArray("filter")

	filterNames := []string{"size", "name"}
	_, searchFilter := utils.ParseFilterParameters(filter, filterNames)

	duParams := operations.NewGetWorkflowDiskUsageParams()
	duParams.SetAccessToken(&token)
	duParams.SetWorkflowIDOrName(workflow)
	additionalParams := operations.GetWorkflowDiskUsageBody{
		Summarize: summarize,
		Search:    searchFilter,
	}
	duParams.SetParameters(additionalParams)

	duResp, err := client.ApiClient().Operations.GetWorkflowDiskUsage(duParams)
	if err != nil {
		fmt.Println("Error: Disk usage could not be retrieved:")
		fmt.Println(err)
		os.Exit(1)
	}

	displayDuPayload(duResp.Payload, humanReadable)
}

func displayDuPayload(p *operations.GetWorkflowDiskUsageOKBody, humanReadable bool) {
	if len(p.DiskUsageInfo) == 0 {
		fmt.Println("Error: No files matching filter criteria.")
		os.Exit(1)
	}

	header := []string{"SIZE", "NAME"}
	var rows [][]any

	for _, diskUsageInfo := range p.DiskUsageInfo {
		if utils.HasAnyPrefix(diskUsageInfo.Name, utils.FilesBlacklist) {
			continue
		}

		var row []any
		if humanReadable {
			row = append(row, diskUsageInfo.Size.HumanReadable)
		} else {
			row = append(row, diskUsageInfo.Size.Raw)
		}
		row = append(row, "."+diskUsageInfo.Name)
		rows = append(rows, row)
	}

	utils.DisplayTable(header, rows)
}
