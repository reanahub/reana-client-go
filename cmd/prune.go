/*
This file is part of REANA.
Copyright (C) 2023 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"

	"github.com/spf13/cobra"
)

const pruneDesc = `
Prune workspace.

The ` + "`prune``" + ` command deletes all the intermediate files of a given workflow that are not present
in the input or output section of the workflow specification.

Examples:

  $ reana-client prune -w myanalysis.42

  $ reana-client prune -w myanalysis.42 --include-inputs
`

const includeInputsFlagDesc = `Delete also the input files of the workflow.
Note that this includes the workflow specification file.`

type pruneOptions struct {
	token          string
	workflow       string
	includeInputs  bool
	includeOutputs bool
}

// newPruneCmd creates a command to prune a workspace.
func newPruneCmd() *cobra.Command {
	o := &pruneOptions{}

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune workspace.",
		Long:  pruneDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
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
	f.BoolVarP(
		&o.includeInputs,
		"include-inputs",
		"i",
		false,
		includeInputsFlagDesc,
	)
	f.BoolVarP(
		&o.includeOutputs,
		"include-outputs",
		"o",
		false,
		"Delete also the output files of the workflow.",
	)
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "", false, "Help for prune")

	return cmd
}

func (o *pruneOptions) run(cmd *cobra.Command) error {
	pruneParams := operations.NewPruneWorkspaceParams()
	pruneParams.SetAccessToken(&o.token)
	pruneParams.SetWorkflowIDOrName(o.workflow)
	pruneParams.SetIncludeInputs(&o.includeInputs)
	pruneParams.SetIncludeOutputs(&o.includeOutputs)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	pruneResp, err := api.Operations.PruneWorkspace(pruneParams)
	if err != nil {
		return err
	}

	displayPrunePayload(cmd, pruneResp.Payload)
	return nil
}

// displayPrunePayload displays the prune payload.
func displayPrunePayload(
	cmd *cobra.Command,
	p *operations.PruneWorkspaceOKBody,
) {
	displayer.DisplayMessage(p.Message, displayer.Success, false, cmd.OutOrStdout())
}
