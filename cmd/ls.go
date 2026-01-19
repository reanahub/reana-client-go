/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/datautils"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/filterer"
	"reanahub/reana-client-go/pkg/formatter"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const lsDesc = `
List workspace files.

The ` + "``ls``" + ` command lists workspace files of a workflow specified by the
environment variable REANA_WORKON or provided as a command-line flag
` + "``--workflow`` or ``-w``." + ` The SOURCE argument is optional and specifies a
pattern matching files and directories.

Examples:

  $ reana-client ls --workflow myanalysis.42

  $ reana-client ls --workflow myanalysis.42 --human-readable

  $ reana-client ls --workflow myanalysis.42 'data/*root*'

  $ reana-client ls --workflow myanalysis.42 --filter name=hello
`

const lsFormatFlagDesc = `Format output according to column titles or column
values. Use <column_name>=<column_value> format.
E.g.display files named data.txt
--format name=data.txt`

const lsFilterFlagDesc = `Filter results to show only files that match certain filtering criteria such as
file name, size or modification date.
Use --filter <column_name>=<column_value> pairs. Available
filters are 'name', 'size' and 'last-modified'.`

type lsOptions struct {
	token         string
	serverURL     string
	workflow      string
	formatFilters []string
	jsonOutput    bool
	displayURLs   bool
	humanReadable bool
	filters       []string
	page          int64
	size          int64
	fileName      string
}

// newLsCmd creates a command to list workspace files.
func newLsCmd() *cobra.Command {
	o := &lsOptions{}

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List workspace files.",
		Long:  lsDesc,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			if len(args) > 0 {
				o.fileName = args[0]
			}
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
	f.StringSliceVar(&o.formatFilters, "format", []string{}, lsFormatFlagDesc)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	f.BoolVar(&o.displayURLs, "url", false, "Get URLs of output files.")
	f.BoolVarP(
		&o.humanReadable,
		"human-readable",
		"h",
		false,
		"Show disk size in human readable format.",
	)
	f.StringSliceVar(&o.filters, "filter", []string{}, lsFilterFlagDesc)
	f.Int64Var(
		&o.page,
		"page",
		1,
		"Results page number (to be used with --size).",
	)
	f.Int64Var(
		&o.size,
		"size",
		0,
		"Number of results per page (to be used with --page).",
	)
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "", false, "Help for ls")

	return cmd
}

func (o *lsOptions) run(cmd *cobra.Command) error {
	header := []string{"name", "size", "last-modified"}

	filters, err := filterer.NewFilters(nil, header, o.filters)
	if err != nil {
		return err
	}
	searchFilter, err := filters.GetJson(header)
	if err != nil {
		return err
	}

	log.Infof("Workflow %s selected", o.workflow)

	lsParams := operations.NewGetFilesParams()
	lsParams.SetAccessToken(&o.token)
	lsParams.SetWorkflowIDOrName(o.workflow)
	lsParams.SetFileName(&o.fileName)
	lsParams.SetSearch(&searchFilter)
	lsParams.SetPage(&o.page)
	if cmd.Flags().Changed("size") {
		lsParams.SetSize(&o.size)
	}

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	lsResp, err := api.Operations.GetFiles(lsParams)
	if err != nil {
		return err
	}

	parsedFormatFilters := formatter.ParseFormatParameters(
		o.formatFilters,
		true,
	)
	if o.displayURLs {
		displayLsURLs(cmd, lsResp.Payload, o.serverURL, o.workflow)
	} else {
		err = displayLsFiles(
			cmd,
			lsResp.Payload,
			header,
			parsedFormatFilters,
			o.jsonOutput,
			o.humanReadable,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func displayLsFiles(
	cmd *cobra.Command,
	p *operations.GetFilesOKBody,
	header []string,
	formatFilters []formatter.FormatFilter,
	jsonOutput bool,
	humanReadable bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := buildLsSeries(col, humanReadable)
		for _, file := range p.Items {
			if datautils.HasAnyPrefix(file.Name, config.FilesBlacklist) {
				continue
			}

			var value any
			switch col {
			case "name":
				value = file.Name
			case "size":
				if humanReadable {
					value = file.Size.HumanReadable
				} else {
					value = int(file.Size.Raw)
				}
			case "last-modified":
				value = file.LastModified
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

func buildLsSeries(col string, humanReadable bool) series.Series {
	if col == "size" && !humanReadable {
		return series.New([]int{}, series.Int, col)
	}
	return series.New([]string{}, series.String, col)
}

func displayLsURLs(
	cmd *cobra.Command,
	p *operations.GetFilesOKBody,
	serverURL string,
	workflow string,
) {
	for _, file := range p.Items {
		fileURL := fmt.Sprintf(
			"%s/api/workflows/%s/workspace/%s",
			serverURL,
			workflow,
			file.Name,
		)
		cmd.Println(fileURL)
	}
}
