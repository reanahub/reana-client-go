/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package open provides the command to open an interactive session inside the workspace.
package open

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/formatter"
	"reanahub/reana-client-go/pkg/validator"

	"github.com/jedib0t/go-pretty/v6/text"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const description = `
Open an interactive session inside the workspace.

The ` + "``open``" + ` command allows to open interactive session processes on top of
the workflow workspace, such as Jupyter notebooks. This is useful to quickly
inspect and analyse the produced files while the workflow is still running.

Examples:

  $ reana-client open -w myanalysis.42 jupyter
`

const imageFlagDesc = `Docker image which will be used to spawn the
interactive session. Overrides the default image
for the selected type.`

type options struct {
	token                  string
	serverURL              string
	workflow               string
	image                  string
	interactiveSessionType string
}

// NewCmd creates a command to open an interactive session inside the workspace.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open an interactive session inside the workspace.",
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			o.interactiveSessionType = config.InteractiveSessionTypes[0]
			if len(args) > 0 {
				o.interactiveSessionType = args[0]
			}
			if err := validator.ValidateChoice(
				o.interactiveSessionType,
				config.InteractiveSessionTypes,
				"interactive-session-type",
			); err != nil {
				return err
			}
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
	f.StringVarP(&o.image, "image", "i", "", imageFlagDesc)

	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
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
	log.Infof("Opening an interactive session on %s", o.workflow)
	openResp, err := api.Operations.OpenInteractiveSession(openParams)
	if err != nil {
		return err
	}

	displayer.DisplayMessage(
		"Interactive session opened successfully",
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)
	sessionURI := formatter.FormatSessionURI(o.serverURL, openResp.Payload.Path, o.token)
	displayer.PrintColorable(sessionURI+"\n", cmd.OutOrStdout(), text.FgGreen)
	cmd.Println("It could take several minutes to start the interactive session.")
	return nil
}
