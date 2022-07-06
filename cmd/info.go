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

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "List cluster general information.",
		Long: `
	List cluster general information.

	The ` + "``info``" + ` command lists general information about the cluster.

	Lists all the available workspaces. It also returns the default workspace
	defined by the admin.

	Examples:

	  $ reana-client info
		`,
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			token, _ := cmd.Flags().GetString("access-token")
			serverURL := os.Getenv("REANA_SERVER_URL")
			validation.ValidateAccessToken(token)
			validation.ValidateServerURL(serverURL)
			info(token, jsonOutput)
		},
	}

	cmd.Flags().BoolP("json", "", false, "Get output in JSON format.")
	cmd.Flags().StringP("access-token", "t", os.Getenv("REANA_ACCESS_TOKEN"), "Access token of the current user.")

	return cmd
}

func info(token string, jsonOutput bool) {
	infoParams := operations.NewInfoParams()
	infoParams.SetAccessToken(token)
	infoResp, err := client.ApiClient.Operations.Info(infoParams)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	p := infoResp.Payload
	if jsonOutput {
		utils.DisplayJsonOutput(p)
	} else {
		response := fmt.Sprintf("List of supported compute backends: %s \n", strings.Join(p.ComputeBackends.Value, ", ")) +
			fmt.Sprintf("Default workspace: %s \n", p.DefaultWorkspace.Value) +
			fmt.Sprintf("List of available workspaces: %s \n", strings.Join(p.WorkspacesAvailable.Value, ", "))

		fmt.Print(response)
	}
}
