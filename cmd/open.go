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

	log "github.com/sirupsen/logrus"
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

type openOptions struct {
	token                  string
	serverURL              string
	workflow               string
	image                  string
	interactiveSessionType string
}

func newOpenCmd() *cobra.Command {
	o := &openOptions{}

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open an interactive session inside the workspace.",
		Long:  openDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.token == "" {
				o.token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			o.serverURL = os.Getenv("REANA_SERVER_URL")
			if o.workflow == "" {
				o.workflow = os.Getenv("REANA_WORKON")
			}
			o.interactiveSessionType = utils.InteractiveSessionTypes[0]
			if len(args) > 0 {
				o.interactiveSessionType = args[0]
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
			if err := validation.ValidateChoice(
				o.interactiveSessionType,
				utils.InteractiveSessionTypes,
				"interactive-session-type",
			); err != nil {
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
	f.StringVarP(&o.image, "image", "i", "", openImageFlagDesc)

	return cmd
}

func (o *openOptions) run(cmd *cobra.Command) error {
	openParams := operations.NewOpenInteractiveSessionParams()
	openParams.SetAccessToken(&o.token)
	openParams.SetWorkflowIDOrName(o.workflow)
	openParams.SetInteractiveSessionType(o.interactiveSessionType)
	openParams.SetInteractiveSessionConfiguration(
		operations.OpenInteractiveSessionBody{Image: o.image},
	)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	log.Infof("Opening an interactive session on '%s'", o.workflow)
	openResp, err := api.Operations.OpenInteractiveSession(openParams)
	if err != nil {
		return fmt.Errorf("interactive session could not be opened:\n%v", err)
	}

	cmd.Println("Interactive session opened successfully")
	cmd.Println(utils.FormatSessionURI(o.serverURL, openResp.Payload.Path, o.token))
	cmd.Println("It could take several minutes to start the interactive session.")
	return nil
}
