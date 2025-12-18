/*
This file is part of REANA.
Copyright (C) 2023 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/formatter"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/spf13/cobra"
)

const shareStatusFormatFlagDesc = `Format output according to column titles or column
values. Use <columm_name>=<column_value> format.`

const shareStatusDesc = `Show with whom a workflow is shared.

The ` + "`share-status`" + ` command allows for checking with whom a workflow is
shared.

Example:

  $ reana-client share-status -w myanalysis.42
`

type shareStatusOptions struct {
	token         string
	workflow      string
	formatFilters []string
	jsonOutput    bool
}

// newShareStatusCmd creates a command to show with whom a workflow is shared.
func newShareStatusCmd() *cobra.Command {
	o := &shareStatusOptions{}

	cmd := &cobra.Command{
		Use:   "share-status",
		Short: "Show with whom a workflow is shared.",
		Long:  shareStatusDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w",
		"",
		`Name or UUID of the workflow. Overrides value of 
	REANA_WORKON environment variable.`,
	)
	f.StringVarP(
		&o.token,
		"access-token",
		"t",
		"",
		"Access token of the current user.",
	)
	f.StringSliceVar(
		&o.formatFilters,
		"format",
		[]string{},
		shareStatusFormatFlagDesc,
	)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")

	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "h", false, "Help for share-status")

	return cmd
}

func (o *shareStatusOptions) run(cmd *cobra.Command) error {
	shareStatusParams := operations.NewGetWorkflowShareStatusParams()
	shareStatusParams.SetAccessToken(&o.token)
	shareStatusParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	shareStatusResp, err := api.Operations.GetWorkflowShareStatus(
		shareStatusParams,
	)
	if err != nil {
		return err
	}

	if len(shareStatusResp.Payload.SharedWith) == 0 {
		displayer.DisplayMessage(
			fmt.Sprintf("Workflow %s is not shared with anyone.", o.workflow),
			displayer.Info,
			false,
			cmd.OutOrStdout(),
		)

		return nil
	}

	parsedFormatFilters := formatter.ParseFormatParameters(
		o.formatFilters,
		true,
	)
	header := []string{"user_email", "valid_until"}

	err = displayShareStatusPayload(
		cmd,
		shareStatusResp.Payload,
		header,
		parsedFormatFilters,
		o.jsonOutput,
	)
	return err
}

func displayShareStatusPayload(
	cmd *cobra.Command,
	payload *operations.GetWorkflowShareStatusOKBody,
	header []string,
	formatFilters []formatter.FormatFilter,
	jsonOutput bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := series.New([]string{}, series.String, col)
		for _, share := range payload.SharedWith {
			var value any

			switch col {
			case "user_email":
				value = share.UserEmail
			case "valid_until":
				if share.ValidUntil != nil {
					value = *share.ValidUntil
				}
			}

			colSeries.Append(value)
		}

		df = df.CBind(dataframe.New(colSeries))
	}

	df, err := formatter.FormatDataFrame(df, formatFilters)
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
