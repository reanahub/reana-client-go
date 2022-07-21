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
		Run: func(cmd *cobra.Command, args []string) {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			workflow, _ := cmd.Flags().GetString("workflow")
			if workflow == "" {
				workflow = os.Getenv("REANA_WORKON")
			}

			validation.ValidateAccessToken(token)
			validation.ValidateWorkflow(workflow)

			close(token, workflow)
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().
		StringP("workflow", "w", "", "Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.")

	return cmd
}

func close(token string, workflow string) {
	closeParams := operations.NewCloseInteractiveSessionParams()
	closeParams.SetAccessToken(&token)
	closeParams.SetWorkflowIDOrName(workflow)

	_, err := client.ApiClient().Operations.CloseInteractiveSession(closeParams)
	if err != nil {
		fmt.Println("Error: Interactive session could not be closed")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Interactive session for workflow", workflow, "was successfully closed")
}
