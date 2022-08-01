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
	"strings"

	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"

	"github.com/spf13/cobra"
)

const infoDesc = `
List cluster general information.

The ` + "``info``" + ` command lists general information about the cluster.

Lists all the available workspaces. It also returns the default workspace
defined by the admin.
`

type infoOptions struct {
	token      string
	serverURL  string
	jsonOutput bool
}

// newInfoCmd creates a command to list cluster general information.
func newInfoCmd() *cobra.Command {
	o := &infoOptions{}

	cmd := &cobra.Command{
		Use:   "info",
		Short: "List cluster general information.",
		Long:  infoDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.token == "" {
				o.token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			o.serverURL = os.Getenv("REANA_SERVER_URL")

			if err := validation.ValidateAccessToken(o.token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(o.serverURL); err != nil {
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
	f.BoolVarP(&o.jsonOutput, "json", "", false, "Get output in JSON format.")

	return cmd
}

func (o *infoOptions) run(cmd *cobra.Command) error {
	infoParams := operations.NewInfoParams()
	infoParams.SetAccessToken(o.token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	infoResp, err := api.Operations.Info(infoParams)
	if err != nil {
		return err
	}

	p := infoResp.Payload
	if o.jsonOutput {
		err := utils.DisplayJsonOutput(p, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		response := fmt.Sprintf("List of supported compute backends: %s \n", strings.Join(p.ComputeBackends.Value, ", ")) +
			fmt.Sprintf("Default workspace: %s \n", p.DefaultWorkspace.Value) +
			fmt.Sprintf("List of available workspaces: %s \n", strings.Join(p.WorkspacesAvailable.Value, ", "))

		cmd.Print(response)
	}
	return nil
}
