/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package ls provides the command to list workspace files.
package ls

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

const description = `
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

const formatFlagDesc = `Format output according to column titles or column
values. Use <column_name>=<column_value> format.
E.g.display files named data.txt
--format name=data.txt`

const filterFlagDesc = `Filter results to show only files that match certain filtering criteria such as
file name, size or modification date.
Use --filter <column_name>=<column_value> pairs. Available
filters are 'name', 'size' and 'last-modified'.`

// Options options to be used in the ls command.
type Options struct {
	Token         string
	ServerURL     string
	Workflow      string
	FormatFilters []string
	JsonOutput    bool
	DisplayURLs   bool
	HumanReadable bool
	Filters       []string
	Page          int64
	Size          int64
	FileName      string
}

// NewCmd creates a command to list workspace files.
func NewCmd() *cobra.Command {
	o := &Options{}

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List workspace files.",
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ServerURL = viper.GetString("server-url")
			if len(args) > 0 {
				o.FileName = args[0]
			}
			return o.Run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.Token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(
		&o.Workflow,
		"workflow",
		"w",
		"",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	f.StringSliceVar(&o.FormatFilters, "format", []string{}, formatFlagDesc)
	f.BoolVar(&o.JsonOutput, "json", false, "Get output in JSON format.")
	f.BoolVar(&o.DisplayURLs, "url", false, "Get URLs of output files.")
	f.BoolVarP(
		&o.HumanReadable,
		"human-readable",
		"h",
		false,
		"Show disk size in human readable format.",
	)
	f.StringSliceVar(&o.Filters, "filter", []string{}, filterFlagDesc)
	f.Int64Var(&o.Page, "page", 1, "Results page number (to be used with --size).")
	f.Int64Var(&o.Size, "size", 0, "Number of results per page (to be used with --page).")
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "", false, "Help for du")

	return cmd
}

// Run runs the ls command with the options provided by o.
func (o *Options) Run(cmd *cobra.Command) error {
	header := []string{"name", "size", "last-modified"}

	filters, err := filterer.NewFilters(nil, header, o.Filters)
	if err != nil {
		return err
	}
	searchFilter, err := filters.GetJson(header)
	if err != nil {
		return err
	}

	log.Infof("Workflow %s selected", o.Workflow)

	lsParams := operations.NewGetFilesParams()
	lsParams.SetAccessToken(&o.Token)
	lsParams.SetWorkflowIDOrName(o.Workflow)
	lsParams.SetFileName(&o.FileName)
	lsParams.SetSearch(&searchFilter)
	lsParams.SetPage(&o.Page)
	if cmd.Flags().Changed("size") {
		lsParams.SetSize(&o.Size)
	}

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	lsResp, err := api.Operations.GetFiles(lsParams)
	if err != nil {
		return err
	}

	parsedFormatFilters := formatter.ParseFormatParameters(o.FormatFilters, true)
	if o.DisplayURLs {
		displayURLs(cmd, lsResp.Payload, o.ServerURL, o.Workflow)
	} else {
		err = displayFiles(
			cmd,
			lsResp.Payload,
			header,
			parsedFormatFilters,
			o.JsonOutput,
			o.HumanReadable,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// displayFiles displays the payload as a file table, according to the output options provided.
func displayFiles(
	cmd *cobra.Command,
	p *operations.GetFilesOKBody,
	header []string,
	formatFilters []formatter.FormatFilter,
	jsonOutput bool,
	humanReadable bool,
) error {
	var df dataframe.DataFrame
	for _, col := range header {
		colSeries := buildSeries(col, humanReadable)
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

// buildSeries build a series.Series to be used as a column for the ls output table.
func buildSeries(col string, humanReadable bool) series.Series {
	if col == "size" && !humanReadable {
		return series.New([]int{}, series.Int, col)
	}
	return series.New([]string{}, series.String, col)
}

// displayURLs displays the payload as a list of URLs.
func displayURLs(
	cmd *cobra.Command,
	p *operations.GetFilesOKBody,
	serverURL string,
	workflow string,
) {
	for _, file := range p.Items {
		fileURL := fmt.Sprintf("%s/api/workflows/%s/workspace/%s", serverURL, workflow, file.Name)
		cmd.Println(fileURL)
	}
}
