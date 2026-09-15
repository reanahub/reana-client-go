/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/specbundle"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const createDesc = `
Create a new workflow.

The ` + "``create``" + ` command allows to create a new workflow from a
reana.yaml specification file. The specification is uploaded as a bundle and
loaded + validated on the REANA server (sandboxed for Snakemake/CWL/Yadage), so
no workflow engines are needed locally.

Examples:

  $ reana-client create

  $ reana-client create -n myanalysis -f myreana.yaml
`

type createOptions struct {
	token          string
	serverURL      string
	file           string
	name           string
	skipValidation bool
}

// newCreateCmd creates a command to create a new workflow.
func newCreateCmd() *cobra.Command {
	o := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new workflow.",
		Long:  createDesc,
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
		"REANA specification file describing the workflow to create. [default=reana.yaml]",
	)
	// The workflow name accepts the same aliases as the Python client:
	// -n/--name and -w/--workflow, all writing to the same value.
	f.StringVarP(
		&o.name,
		"name",
		"n",
		"",
		`Optional name of the workflow. If not provided, a name will be generated.`,
	)
	f.StringVarP(
		&o.name,
		"workflow",
		"w",
		"",
		`Alias for --name.`,
	)
	f.BoolVar(
		&o.skipValidation,
		"skip-validation",
		false,
		"If set, the specification file is not validated before "+
			"being submitted to the REANA server.",
	)

	// The workflow name is optional (the server generates one when omitted), so
	// the `--workflow` alias must not trigger the root's required-workflow check.
	if err := f.SetAnnotation("workflow", "properties", []string{"optional"}); err != nil {
		log.Debugf("Failed to set workflow annotation: %s", err.Error())
	}

	return cmd
}

func (o *createOptions) run(cmd *cobra.Command) error {
	workflowName, err := o.create(cmd)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), workflowName)
	return nil
}

// create creates a workflow and returns the canonical name assigned by the server.
func (o *createOptions) create(cmd *cobra.Command) (string, error) {
	out := cmd.OutOrStdout()

	if o.skipValidation {
		displayer.DisplayMessage(
			"The specification is now always validated server-side; the "+
				"`--skip-validation` flag is ignored.",
			displayer.Warning,
			false,
			out,
		)
	}

	if err := validateWorkflowNameForCreate(o.name); err != nil {
		return "", err
	}

	// Refuse an incompatible server before any bundle is built or uploaded.
	api, err := bundleCapableAPIClient(o.token)
	if err != nil {
		return "", err
	}

	members, err := specbundle.Gather(o.file)
	if err != nil {
		return "", err
	}
	bundle, err := specbundle.Archive(members)
	if err != nil {
		return "", err
	}
	defer func() { _ = bundle.Close() }()

	params := operations.NewCreateWorkflowParamsWithTimeout(
		controlOperationTimeout,
	)
	params.SetBundle(bundle)
	params.SetWorkflowName(o.name)
	okResponse, createdResponse, err := api.Operations.CreateWorkflow(
		params,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("cannot create workflow: %w", err)
	}

	if okResponse != nil {
		return "", fmt.Errorf(
			"server returned a legacy create response without a workflow name: %s",
			okResponse.GetPayload().Message,
		)
	}
	payload := createdResponse.GetPayload()
	for _, warning := range payload.ValidationWarnings {
		if warning != nil {
			displayer.DisplayMessage(
				warning.Message,
				displayer.Warning,
				true,
				out,
			)
		}
	}
	return payload.WorkflowName, nil
}

// validateWorkflowNameForCreate rejects, client-side, a name the server would
// reject, matching the Python client's early guardrails. An empty name is
// allowed (the server generates one); a name that contains the reserved "."
// separator or is a valid UUIDv4 is refused before contacting the server.
func validateWorkflowNameForCreate(name string) error {
	if name == "" {
		return nil
	}
	if strings.Contains(name, ".") {
		return fmt.Errorf(
			"workflow name %s contains illegal character '.'",
			name,
		)
	}
	if parsed, err := uuid.Parse(name); err == nil && parsed.Version() == 4 {
		return errors.New("workflow name cannot be a valid UUIDv4")
	}
	return nil
}
