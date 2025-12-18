/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/formatter"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/spf13/cobra"
)

const retentionRulesListDesc = `
List the retention rules for a workflow.

Example:

	 $ reana-client retention-rules-list -w myanalysis.42
`

const retentionRulesListFormatFlagDesc = `Format output according to column titles or column
values. Use <columm_name>=<column_value> format.
E.g. display pattern and status of active retention rules
--format workspace_files,status=active.`

type retentionRulesListOptions struct {
	token         string
	workflow      string
	jsonOutput    bool
	formatFilters []string
}

// newRetentionRulesListCmd creates a command to list retention rules.
func newRetentionRulesListCmd() *cobra.Command {
	o := &retentionRulesListOptions{}

	cmd := &cobra.Command{
		Use:   "retention-rules-list",
		Short: "List the retention rules for a workflow.",
		Long:  retentionRulesListDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(
		&o.token,
		"access-token",
		"t",
		"",
		"Access token of the current user.",
	)
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w",
		"",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	f.StringSliceVar(
		&o.formatFilters,
		"format",
		[]string{},
		retentionRulesListFormatFlagDesc,
	)

	return cmd
}

func (o *retentionRulesListOptions) run(cmd *cobra.Command) error {
	retentionRulesParams := operations.NewGetWorkflowRetentionRulesParams()
	retentionRulesParams.SetAccessToken(&o.token)
	retentionRulesParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	retentionRulesResp, err := api.Operations.GetWorkflowRetentionRules(
		retentionRulesParams,
	)
	if err != nil {
		return err
	}

	df := buildRetentionRulesDataFrame(retentionRulesResp)
	df = df.Arrange(dataframe.Sort("retention_days"))

	parsedFormatFilters := formatter.ParseFormatParameters(
		o.formatFilters,
		true,
	)
	df, err = formatter.FormatDataFrame(df, parsedFormatFilters)
	if err != nil {
		return err
	}

	if o.jsonOutput {
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

func buildRetentionRulesDataFrame(
	response *operations.GetWorkflowRetentionRulesOK,
) dataframe.DataFrame {
	workspaceFilesSeries := series.New(
		[]string{},
		series.String,
		"workspace_files",
	)
	retentionDaysSeries := series.New([]int{}, series.Int, "retention_days")
	applyOnSeries := series.New([]string{}, series.String, "apply_on")
	statusSeries := series.New([]string{}, series.String, "status")

	for _, rule := range response.Payload.RetentionRules {
		workspaceFilesSeries.Append(rule.WorkspaceFiles)
		retentionDaysSeries.Append(int(rule.RetentionDays))
		if rule.ApplyOn == nil {
			applyOnSeries.Append(nil)
		} else {
			applyOnSeries.Append(*rule.ApplyOn)
		}
		statusSeries.Append(rule.Status)
	}

	return dataframe.New(
		workspaceFilesSeries,
		retentionDaysSeries,
		applyOnSeries,
		statusSeries,
	)
}
