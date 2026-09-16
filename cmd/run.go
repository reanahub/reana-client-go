/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"

	"github.com/reanahub/reana-client-go/pkg/displayer"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const runDesc = `
Create, upload, and start a new workflow.

The ` + "``run``" + ` command is a shortcut that creates a workflow, uploads
the input files declared in its specification, and starts it.

Examples:

  $ reana-client run -w myanalysis

  $ reana-client run -w myanalysis -p events=100 -o CACHE=off
`

type runOptions struct {
	token          string
	serverURL      string
	file           string
	name           string
	skipValidation bool
	parameters     map[string]string
	options        map[string]string
	follow         bool
}

// newRunCmd creates a command to create, upload, and start a workflow.
func newRunCmd() *cobra.Command {
	o := &runOptions{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Create, upload, and start a new workflow.",
		Long:  runDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
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
		&o.file,
		"file",
		"f",
		"reana.yaml",
		"REANA specification file describing the workflow to execute. [default=reana.yaml]",
	)
	f.StringVarP(
		&o.name,
		"name",
		"n",
		"",
		"Optional name of the workflow. If not provided, a name will be generated.",
	)
	f.StringVarP(&o.name, "workflow", "w", "", "Alias for --name.")
	f.BoolVar(
		&o.skipValidation,
		"skip-validation",
		false,
		"If set, the specification file is not validated before being submitted to the REANA server.",
	)
	f.StringToStringVarP(
		&o.parameters,
		"parameter",
		"p",
		map[string]string{},
		"Additional input parameters to override values from reana.yaml.",
	)
	f.StringToStringVarP(
		&o.options,
		"option",
		"o",
		map[string]string{},
		"Additional operational options for the workflow execution.",
	)
	f.BoolVar(
		&o.follow,
		"follow",
		false,
		"If set, follows the execution of the workflow until termination.",
	)

	if err := f.SetAnnotation("workflow", "properties", []string{"optional"}); err != nil {
		log.Debugf("Failed to set workflow annotation: %s", err.Error())
	}

	return cmd
}

func (o *runOptions) run(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	displayer.DisplayMessage(
		"Creating a workflow...",
		displayer.Info,
		false,
		out,
	)
	workflow, err := (&createOptions{
		token:          o.token,
		serverURL:      o.serverURL,
		file:           o.file,
		name:           o.name,
		skipValidation: o.skipValidation,
	}).create(cmd)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, workflow)

	displayer.DisplayMessage("Uploading files...", displayer.Info, false, out)
	if err := (&uploadOptions{token: o.token, workflow: workflow}).run(cmd, nil); err != nil {
		return err
	}

	displayer.DisplayMessage("Starting workflow...", displayer.Info, false, out)
	return (&startOptions{
		token:      o.token,
		serverURL:  o.serverURL,
		workflow:   workflow,
		parameters: o.parameters,
		options:    o.options,
		follow:     o.follow,
	}).run(cmd)
}
