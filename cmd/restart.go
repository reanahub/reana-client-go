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
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/validator"
	"reanahub/reana-client-go/pkg/workflows"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
)

const restartDesc = `
Restart previously run workflow.

The ` + "``restart``" + ` command allows to restart a previous workflow on the same
workspace.

Note that workflow restarting can be used in a combination with operational
options ` + "``FROM``" + ` and ` + "``TARGET``" + `. You can also pass a modified workflow
specification with ` + "``-f``" + ` or ` + "``--file``" + ` flag.

You can furthermore use modified input prameters using ` + "``-p``" + ` or
` + "``--parameters``" + ` flag and by setting additional operational options using
` + "``-o``" + ` or ` + "``--options``" + `.  The input parameters and operational options can
be repetitive.

Examples:

  $ reana-client restart -w myanalysis.42 -p sleeptime=10 -p myparam=4

  $ reana-client restart -w myanalysis.42 -p myparam=myvalue

  $ reana-client restart -w myanalysis.42 -o TARGET=gendata

  $ reana-client restart -w myanalysis.42 -o FROM=fitdata
`

type restartOptions struct {
	token      string
	workflow   string
	parameters map[string]string
	options    map[string]string
	file       string
}

// newRestartCmd creates a command to restart previously run workflow.
func newRestartCmd() *cobra.Command {
	o := &restartOptions{}

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart previously run workflow.",
		Long:  restartDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.file != "" {
				if err := validator.ValidateFile(o.file); err != nil {
					return fmt.Errorf(
						"invalid value for '--file': %s",
						err.Error(),
					)
				}
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
	f.StringToStringVarP(
		&o.parameters,
		"parameter",
		"p",
		map[string]string{},
		`Additional input parameters to override original ones from reana.yaml.
E.g. -p myparam1=myval1 -p myparam2=myval2.`,
	)
	f.StringToStringVarP(
		&o.options,
		"option",
		"o",
		map[string]string{},
		`Additional operational options for the workflow execution.
E.g. CACHE=off. (workflow engine - serial)
E.g. --debug (workflow engine - cwl)`,
	)
	f.StringVarP(
		&o.file,
		"file",
		"f", "reana.yaml",
		"REANA specification file describing the workflow to execute.",
	)
	return cmd
}

func (o *restartOptions) run(cmd *cobra.Command) error {
	api, err := client.ApiClient()
	if err != nil {
		return err
	}

	// TODO: support ReanaSpecification file upload

	if len(o.parameters) > 0 || len(o.options) > 0 {
		o.options, o.parameters, err = validateStartOptionsAndParams(
			api,
			o.token, o.workflow, o.options, o.parameters,
			cmd.OutOrStdout(),
		)
		if err != nil {
			return err
		}
	}

	startParams := operations.NewStartWorkflowParams()
	startParams.SetAccessToken(&o.token)
	startParams.SetWorkflowIDOrName(o.workflow)
	startParams.SetParameters(operations.StartWorkflowBody{
		InputParameters:    o.parameters,
		OperationalOptions: o.options,
		Restart:            true,
	})
	startResp, err := api.Operations.StartWorkflow(startParams)
	if err != nil {
		return err
	}

	currentStatus := startResp.Payload.Status
	statusMsg, err := workflows.StatusChangeMessage(o.workflow, currentStatus)
	if err != nil {
		return err
	}
	if slices.Contains(
		[]string{"pending", "queued", "running"},
		currentStatus,
	) {
		displayer.DisplayMessage(
			statusMsg,
			displayer.Success,
			false,
			cmd.OutOrStdout(),
		)
	} else {
		return errors.New(statusMsg)
	}

	return nil
}
