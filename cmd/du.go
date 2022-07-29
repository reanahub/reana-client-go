/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
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

type duOptions struct {
	token         string
	serverURL     string
	workflow      string
	summarize     bool
	humanReadable bool
	filter        []string
}

func newDuCmd() *cobra.Command {
	o := &duOptions{}

	cmd := &cobra.Command{
		Use:   "du",
		Short: "Get workspace disk usage.",
		Long:  duDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.token == "" {
				o.token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			o.serverURL = os.Getenv("REANA_SERVER_URL")
			if o.workflow == "" {
				o.workflow = os.Getenv("REANA_WORKON")
			}

			if err := validation.ValidateAccessToken(o.token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(o.serverURL); err != nil {
				return err
			}
			if err := validation.ValidateWorkflow(o.workflow); err != nil {
				return err
			}
			if err := o.run(cmd); err != nil {
				return err
			}
			return nil
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
	f.BoolVarP(&o.summarize, "summarize", "s", false, "Display total.")
	f.BoolVarP(
		&o.humanReadable,
		"human-readable",
		"r",
		false,
		"Show disk size in human readable format.",
	)
	f.StringSliceVar(&o.filter, "filter", []string{}, duFilterFlagDesc)

	return cmd
}

func (o *duOptions) run(cmd *cobra.Command) error {
	filterNames := []string{"size", "name"}
	_, searchFilter, err := utils.ParseFilterParameters(o.filter, filterNames)
	if err != nil {
		return err
	}

	duParams := operations.NewGetWorkflowDiskUsageParams()
	duParams.SetAccessToken(&o.token)
	duParams.SetWorkflowIDOrName(o.workflow)
	additionalParams := operations.GetWorkflowDiskUsageBody{
		Summarize: o.summarize,
		Search:    searchFilter,
	}
	duParams.SetParameters(additionalParams)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	duResp, err := api.Operations.GetWorkflowDiskUsage(duParams)
	if err != nil {
		return err
	}

	err = displayDuPayload(cmd, duResp.Payload, o.humanReadable)
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
