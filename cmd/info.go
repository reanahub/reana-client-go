/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"

	"github.com/spf13/cobra"
)

type Info struct {
	ComputeBackends struct {
		Title string   `json:"title"`
		Value []string `json:"value"`
	} `json:"compute_backends"`
	DefaultWorkspace struct {
		Title string `json:"title"`
		Value string `json:"value"`
	} `json:"default_workspace"`
	AvailableWorkspaces struct {
		Title string   `json:"title"`
		Value []string `json:"value"`
	} `json:"workspaces_available"`
}

var infoCmd = &cobra.Command{
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
		info(token, serverURL, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	infoCmd.Flags().BoolP("json", "", false, "Get output in JSON format.")
	infoCmd.Flags().StringP("access-token", "t", os.Getenv("REANA_ACCESS_TOKEN"), "Access token of the current user.")
}

func info(token string, serverURL string, jsonOutput bool) {
	respBytes := utils.NewRequest(token, serverURL, "/api/info")
	i := Info{}

	if err := json.Unmarshal(respBytes, &i); err != nil {
		fmt.Printf("Could not unmarshal reponseBytes. %v", err)
	}
	if jsonOutput {
		utils.DisplayJsonOutput(i)
	} else {
		response := fmt.Sprintf("List of supported compute backends: %s \n", strings.Join(i.ComputeBackends.Value, ", ")) +
			fmt.Sprintf("Default workspace: %s \n", i.DefaultWorkspace.Value) +
			fmt.Sprintf("List of available workspaces: %s \n", strings.Join(i.AvailableWorkspaces.Value, ", "))

		fmt.Print(response)
	}
}
