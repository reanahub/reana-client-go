/*
This file is part of REANA.
Copyright (C) 2022, 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const closeDesc = `
Close an interactive session.

The ` + "``close``" + ` command allows to shut down any interactive sessions that you
may have running. You would typically use this command after you finished
exploring data in the Jupyter notebook and after you have transferred any
code created in your interactive session.

Examples:

  $ reana-client close -w myanalysis.42
`

type closeOptions struct {
	token    string
	workflow string
}

// newCloseCmd creates a command to close an interactive session.
func newCloseCmd() *cobra.Command {
	o := &closeOptions{}

	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close an interactive session.",
		Long:  closeDesc,
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

	return cmd
}

func (o *closeOptions) run(cmd *cobra.Command) error {
	closeParams := operations.NewCloseInteractiveSessionParams()
	closeParams.SetAccessToken(&o.token)
	closeParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	log.Infof("Closing an interactive session on %s", o.workflow)
	_, err = api.Operations.CloseInteractiveSession(closeParams)
	if err != nil {
		return err
	}

	displayer.DisplayMessage(
		fmt.Sprintf(
			"Interactive session for workflow %s was successfully closed",
			o.workflow,
		),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)
	return nil
}
