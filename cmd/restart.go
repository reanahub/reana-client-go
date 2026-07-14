/*
This file is part of REANA.
Copyright (C) 2022, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/specbundle"
	"reanahub/reana-client-go/pkg/validator"
	"reanahub/reana-client-go/pkg/workflows"

	"go.yaml.in/yaml/v3"
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

The ` + "``-f``" + `/` + "``--file``" + ` flag replaces only the *specification* (input
parameters, operational options, and workflow type/definition metadata). The
workflow **source files** (Snakefiles, CWL files, rules, ...) are reused from
the existing workspace: a restart does not re-upload them. If the replacement
specification references new or changed workflow source, upload those files to
the workspace first; otherwise validation fails naming the missing file.

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
		"f", "",
		"REANA specification file with the replacement specification (input "+
			"parameters, operational options, workflow type/definition metadata). "+
			"Workflow source files (Snakefiles, CWL, rules) are reused from the "+
			"existing workspace and are not re-uploaded.",
	)
	return cmd
}

func (o *restartOptions) run(cmd *cobra.Command) error {
	// Only a replacement restart needs the bundle protocol: it uploads the raw
	// specification to the multipart /restart operation, which released servers
	// do not have. A restart without --file stays on the compatible /start.
	var api *client.API
	var err error
	if o.file != "" {
		api, err = bundleCapableAPIClient(o.token)
	} else {
		var authenticated *client.AuthenticatedClient
		authenticated, err = client.ControlAPIClient(o.token)
		if err == nil {
			api = authenticated.API
		}
	}
	if err != nil {
		return err
	}

	var replacementSpecification *restartSpecification
	if o.file != "" {
		if len(o.parameters) > 0 || len(o.options) > 0 {
			replacementSpecification, err = readRestartSpecification(o.file)
			if err != nil {
				return err
			}
		}
	}

	if len(o.parameters) > 0 || len(o.options) > 0 {
		if replacementSpecification != nil {
			o.options, o.parameters, err = validateOptionsAndParams(
				replacementSpecification.Workflow.Type,
				replacementSpecification.Inputs.Parameters,
				o.options,
				o.parameters,
				cmd.OutOrStdout(),
			)
		} else {
			o.options, o.parameters, err = validateStartOptionsAndParams(
				api,
				o.token, o.workflow, o.options, o.parameters,
				cmd.OutOrStdout(),
			)
		}
		if err != nil {
			return err
		}
	}
	var currentStatus string
	if o.file != "" {
		replacement, openErr := specbundle.OpenSpecification(o.file)
		if openErr != nil {
			return fmt.Errorf("file %s could not be read: %s", o.file, openErr)
		}
		defer func() { _ = replacement.Close() }()
		parameters, marshalErr := json.Marshal(map[string]any{
			"input_parameters":    o.parameters,
			"operational_options": o.options,
		})
		if marshalErr != nil {
			return marshalErr
		}
		restartParams := operations.NewRestartWorkflowParamsWithTimeout(
			controlOperationTimeout,
		)
		restartParams.SetWorkflowIDOrName(o.workflow)
		restartParams.SetReplacement(replacement)
		parametersJSON := string(parameters)
		restartParams.SetParameters(&parametersJSON)
		restartResp, restartErr := api.Operations.RestartWorkflow(
			restartParams,
			nil,
		)
		if restartErr != nil {
			return restartErr
		}
		for _, warning := range restartResp.Payload.ValidationWarnings {
			if warning != nil {
				displayer.DisplayMessage(
					warning.Message, displayer.Warning, true, cmd.OutOrStdout(),
				)
			}
		}
		currentStatus = restartResp.Payload.Status
	} else {
		startParams := operations.NewStartWorkflowParamsWithTimeout(
			controlOperationTimeout,
		)
		startParams.SetWorkflowIDOrName(o.workflow)
		startParams.SetParameters(operations.StartWorkflowBody{
			InputParameters:    o.parameters,
			OperationalOptions: o.options,
			Restart:            true,
		})
		startResp, startErr := api.Operations.StartWorkflow(startParams, nil)
		if startErr != nil {
			return startErr
		}
		displayStartValidationWarnings(
			startResp.Payload.ValidationWarnings,
			cmd.OutOrStdout(),
		)
		currentStatus = startResp.Payload.Status
	}
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

// restartSpecification contains only the fields needed for lightweight local
// override validation. The server remains authoritative for full validation.
type restartSpecification struct {
	Inputs struct {
		Parameters map[string]any `yaml:"parameters"`
	} `yaml:"inputs"`
	Workflow struct {
		Type string `yaml:"type"`
	} `yaml:"workflow"`
}

func readRestartSpecification(path string) (*restartSpecification, error) {
	data, err := specbundle.ReadSpecification(path)
	if err != nil {
		return nil, fmt.Errorf(
			"file %s could not be read: %s",
			path,
			err.Error(),
		)
	}
	var spec restartSpecification
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf(
			"file %s is not a valid YAML specification: %s",
			path,
			err.Error(),
		)
	}
	return &spec, nil
}
