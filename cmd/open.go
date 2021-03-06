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
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"

	"github.com/spf13/cobra"
)

const openDesc = `
Open an interactive session inside the workspace.

The ` + "``open``" + ` command allows to open interactive session processes on top of
the workflow workspace, such as Jupyter notebooks. This is useful to quickly
inspect and analyse the produced files while the workflow is still running.

Examples:

  $ reana-client open -w myanalysis.42 jupyter
`

const openImageFlagDesc = `
Docker image which will be used to spawn the
interactive session. Overrides the default image
for the selected type.
`

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open an interactive session inside the workspace.",
		Long:  openDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")
			workflow, _ := cmd.Flags().GetString("workflow")
			if workflow == "" {
				workflow = os.Getenv("REANA_WORKON")
			}
			interactiveSessionType := utils.InteractiveSessionTypes[0]
			if len(args) > 0 {
				interactiveSessionType = args[0]
			}

			if err := validation.ValidateAccessToken(token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(serverURL); err != nil {
				return err
			}
			if err := validation.ValidateWorkflow(workflow); err != nil {
				return err
			}
			if err := validation.ValidateArgChoice(
				interactiveSessionType,
				utils.InteractiveSessionTypes,
				"interactive-session-type",
			); err != nil {
				return err
			}
			if err := open(cmd, token, serverURL, workflow, interactiveSessionType); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().
		StringP("workflow", "w", "", "Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.")
	cmd.Flags().StringP("image", "i", "", openImageFlagDesc)

	return cmd
}

func open(
	cmd *cobra.Command,
	token string,
	serverURL string,
	workflow string,
	interactiveSessionType string,
) error {
	image, _ := cmd.Flags().GetString("image")

	openParams := operations.NewOpenInteractiveSessionParams()
	openParams.SetAccessToken(&token)
	openParams.SetWorkflowIDOrName(workflow)
	openParams.SetInteractiveSessionType(interactiveSessionType)
	openParams.SetInteractiveSessionConfiguration(
		operations.OpenInteractiveSessionBody{Image: image},
	)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	openResp, err := api.Operations.OpenInteractiveSession(openParams)
	if err != nil {
		return fmt.Errorf("interactive session could not be opened:\n%v", err)
	}

	cmd.Println("Interactive session opened successfully")
	cmd.Println(utils.FormatSessionURI(serverURL, openResp.Payload.Path, token))
	cmd.Println("It could take several minutes to start the interactive session.")
	return nil
}
