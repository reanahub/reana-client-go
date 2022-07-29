/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/validation"

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
	token     string
	serverURL string
	workflow  string
}

func newCloseCmd() *cobra.Command {
	o := &closeOptions{}

	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close an interactive session.",
		Long:  closeDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.token == "" {
				o.token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			o.serverURL = os.Getenv("REANA_SERVER_URL")
			if o.workflow == "" {
				o.workflow = os.Getenv("REANA_WORKON")
			}

			if err := validation.ValidateAccessToken(o.token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(o.serverURL); err != nil {
				return err
			}
			if err := validation.ValidateWorkflow(o.workflow); err != nil {
				return err
			}
			if err := o.run(cmd); err != nil {
				return err
			}
			return nil
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

func (o *closeOptions) run(cmd *cobra.Command) error {
	closeParams := operations.NewCloseInteractiveSessionParams()
	closeParams.SetAccessToken(&o.token)
	closeParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	log.Infof("Closing an interactive session on '%s'", o.workflow)
	_, err = api.Operations.CloseInteractiveSession(closeParams)
	if err != nil {
		return fmt.Errorf("interactive session could not be closed:\n%v", err)
	}

	cmd.Println("Interactive session for workflow", o.workflow, "was successfully closed")
	return nil
}
