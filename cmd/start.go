/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"io"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/validator"
	"reanahub/reana-client-go/pkg/workflows"
	"time"

	"github.com/spf13/viper"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
)

const startDesc = `
Start previously created workflow.

The ` + "``start``" + ` command allows to start previously created workflow. The
workflow execution can be further influenced by passing input prameters
using ` + "``-p``" + ` or ` + "``--parameters``" + ` flag and by setting additional operational
options using ` + "``-o``" + ` or ` + "``--options``" + `.The input parameters and operational
options can be repetitive. For example, to disable caching for the Serial
workflow engine, you can set ` + "``-o CACHE=off``" + `.

Examples:

$ reana-client start -w myanalysis.42 -p sleeptime=10 -p myparam=4

$ reana-client start -w myanalysis.42 -p myparam1=myvalue1 -o CACHE=off
`

type startOptions struct {
	token      string
	serverURL  string
	workflow   string
	parameters map[string]string
	options    map[string]string
	follow     bool
}

// newStartCmd creates a command to start previously created workflow.
func newStartCmd(api *client.API, viper *viper.Viper) *cobra.Command {
	o := &startOptions{}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start previously created workflow.",
		Long:  startDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			return o.run(cmd, api)
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
	f.BoolVar(
		&o.follow,
		"follow",
		false,
		"If set, follows the execution of the workflow until termination.",
	)

	return cmd
}

func (o *startOptions) run(cmd *cobra.Command, api *client.API) error {
	if len(o.parameters) > 0 || len(o.options) > 0 {
		var err error
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
	displayer.DisplayMessage(
		statusMsg,
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)

	if o.follow {
		err = followWorkflowExecution(cmd, api, currentStatus, o.token, o.serverURL, o.workflow)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateStartOptionsAndParams gets the workflow parameters from the server and validates the options and params provided.
// For operations options, it returns an error if any of them aren't valid. Translated options if necessary.
// For input parameters, simply displays errors if any and continues execution.
func validateStartOptionsAndParams(
	api *client.API,
	token, workflow string,
	options, inputParams map[string]string,
	out io.Writer,
) (validatedOptions map[string]string, validatedParams map[string]string, err error) {
	params := operations.NewGetWorkflowParametersParams()
	params.SetAccessToken(&token)
	params.SetWorkflowIDOrName(workflow)
	paramsResp, err := api.Operations.GetWorkflowParameters(params)
	if err != nil {
		return nil, nil, err
	}

	validatedOptions, err = validator.ValidateOperationalOptions(paramsResp.Payload.Type, options)
	if err != nil {
		return nil, nil, err
	}

	validatedParams, errorList := validator.ValidateInputParameters(
		inputParams,
		paramsResp.Payload.Parameters,
	)
	for _, err := range errorList {
		displayer.DisplayMessage(err.Error(), displayer.Error, false, out)
	}
	return validatedOptions, validatedParams, nil
}

// followWorkflowExecution follow the execution of the workflow, by calling the GetStatus endpoint periodically.
// The interval used for the requests is dictated by config.CheckInterval.
// If the workflow finishes successfully, this calls the ls command to display the workflow files' URLs.
func followWorkflowExecution(
	cmd *cobra.Command,
	api *client.API,
	currentStatus string,
	token, serverURL, workflow string,
) error {
	for slices.Contains([]string{"pending", "queued", "running"}, currentStatus) {
		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
		status, err := workflows.GetStatus(api, token, workflow)
		if err != nil {
			return err
		}
		currentStatus = status.Status

		statusMsg, err := workflows.StatusChangeMessage(workflow, currentStatus)
		if err != nil {
			return err
		}
		displayer.DisplayMessage(
			statusMsg,
			displayer.Success,
			false,
			cmd.OutOrStdout(),
		)

		if currentStatus == "finished" {
			displayer.DisplayMessage(
				"Listing workflow output files...",
				displayer.Info,
				false,
				cmd.OutOrStdout(),
			)
			lsParams := lsOptions{
				token:       token,
				serverURL:   serverURL,
				workflow:    workflow,
				displayURLs: true,
				page:        1,
			}
			err = lsParams.run(cmd, api)
			if err != nil {
				return err
			}
		} else if slices.Contains([]string{"deleted", "failed", "stopped"}, currentStatus) {
			return errors.New("the workflow did not finish")
		}
	}
	return nil
}
