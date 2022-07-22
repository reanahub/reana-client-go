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

func newCloseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close an interactive session.",
		Long:  closeDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			workflow, _ := cmd.Flags().GetString("workflow")
			if workflow == "" {
				workflow = os.Getenv("REANA_WORKON")
			}

			if err := validation.ValidateAccessToken(token); err != nil {
				return err
			}
			if err := validation.ValidateWorkflow(workflow); err != nil {
				return err
			}
			if err := close(cmd, token, workflow); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().
		StringP("workflow", "w", "", "Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.")

	return cmd
}

func close(cmd *cobra.Command, token string, workflow string) error {
	closeParams := operations.NewCloseInteractiveSessionParams()
	closeParams.SetAccessToken(&token)
	closeParams.SetWorkflowIDOrName(workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	_, err = api.Operations.CloseInteractiveSession(closeParams)
	if err != nil {
		return fmt.Errorf("interactive session could not be closed:\n%v", err)
	}

	cmd.Println("Interactive session for workflow", workflow, "was successfully closed")
	return nil
}
