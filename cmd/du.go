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
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")
			workflow, _ := cmd.Flags().GetString("workflow")
			if workflow == "" {
				workflow = os.Getenv("REANA_WORKON")
			}

			if err := validation.ValidateAccessToken(token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(serverURL); err != nil {
				return err
			}
			if err := validation.ValidateWorkflow(workflow); err != nil {
				return err
			}
			if err := du(cmd, token, workflow); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().
		StringP("workflow", "w", "", "Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.")
	cmd.Flags().BoolP("summarize", "s", false, "Display total.")
	cmd.Flags().BoolP("human-readable", "r", false, "Show disk size in human readable format.")
	cmd.Flags().StringSlice("filter", []string{}, duFilterFlagDesc)

	return cmd
}

func du(cmd *cobra.Command, token string, workflow string) error {
	summarize, _ := cmd.Flags().GetBool("summarize")
	humanReadable, _ := cmd.Flags().GetBool("human-readable")
	filter, _ := cmd.Flags().GetStringSlice("filter")

	filterNames := []string{"size", "name"}
	_, searchFilter, err := utils.ParseFilterParameters(filter, filterNames)
	if err != nil {
		return err
	}

	duParams := operations.NewGetWorkflowDiskUsageParams()
	duParams.SetAccessToken(&token)
	duParams.SetWorkflowIDOrName(workflow)
	additionalParams := operations.GetWorkflowDiskUsageBody{
		Summarize: summarize,
		Search:    searchFilter,
	}
	duParams.SetParameters(additionalParams)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	duResp, err := api.Operations.GetWorkflowDiskUsage(duParams)
	if err != nil {
		return fmt.Errorf("disk usage could not be retrieved:\n%v", err)
	}

	err = displayDuPayload(cmd, duResp.Payload, humanReadable)
	if err != nil {
		return err
	}
	return nil
}

func displayDuPayload(
	cmd *cobra.Command,
	p *operations.GetWorkflowDiskUsageOKBody,
	humanReadable bool,
) error {
	if len(p.DiskUsageInfo) == 0 {
		return errors.New("no files matching filter criteria")
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

	utils.DisplayTable(header, rows, cmd.OutOrStdout())
	return nil
}
