/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"

	"github.com/spf13/cobra"
)

const mvDesc = `
Move files within workspace.

The ` + "``mv``" + ` command allows to move the files within workspace.

Examples:

	$ reana-client mv data/input.txt input/input.txt
`

type mvOptions struct {
	token    string
	workflow string
	source   string
	target   string
}

// newMvCmd creates a command to move files within workspace.
func newMvCmd() *cobra.Command {
	o := &mvOptions{}

	cmd := &cobra.Command{
		Use:   "mv",
		Short: "Move files within workspace.",
		Long:  mvDesc,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.source = args[0]
			o.target = args[1]
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

	return cmd
}

func (o *mvOptions) run(cmd *cobra.Command) error {
	mvParams := operations.NewMoveFilesParams()
	mvParams.SetAccessToken(&o.token)
	mvParams.SetWorkflowIDOrName(o.workflow)
	mvParams.SetSource(o.source)
	mvParams.SetTarget(o.target)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	_, err = api.Operations.MoveFiles(mvParams)
	if err != nil {
		return err
	}

	displayer.DisplayMessage(
		fmt.Sprintf("%s was successfully moved to %s", o.source, o.target),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)

	return nil
}
