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

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "List cluster general information.",
		Long:  infoDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")

			if err := validation.ValidateAccessToken(token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(serverURL); err != nil {
				return err
			}
			if err := info(cmd, token); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolP("json", "", false, "Get output in JSON format.")
	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")

	return cmd
}

func info(cmd *cobra.Command, token string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	infoParams := operations.NewInfoParams()
	infoParams.SetAccessToken(token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	infoResp, err := api.Operations.Info(infoParams)
	if err != nil {
		return err
	}

	p := infoResp.Payload
	if jsonOutput {
		err := utils.DisplayJsonOutput(p)
		if err != nil {
			return err
		}
	} else {
		response := fmt.Sprintf("List of supported compute backends: %s \n", strings.Join(p.ComputeBackends.Value, ", ")) +
			fmt.Sprintf("Default workspace: %s \n", p.DefaultWorkspace.Value) +
			fmt.Sprintf("List of available workspaces: %s \n", strings.Join(p.WorkspacesAvailable.Value, ", "))

		fmt.Print(response)
	}
	return nil
}
