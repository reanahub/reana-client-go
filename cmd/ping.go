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

	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"

	"github.com/spf13/cobra"
)

type Ping struct {
	Email         string `json:"email"`
	FullName      string `json:"full_name"`
	Username      string `json:"username"`
	ServerVersion string `json:"reana_server_version"`
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Check connection to REANA server.",
	Long: `
Check connection to REANA server.

The ` + "``ping``" + ` command allows to test connection to REANA server.

Examples:

  $ reana-client ping
	`,
	Run: func(cmd *cobra.Command, args []string) {
		token, _ := cmd.Flags().GetString("access-token")
		serverURL := os.Getenv("REANA_SERVER_URL")
		validation.ValidateAccessToken(token)
		validation.ValidateServerURL(serverURL)
		ping(token, serverURL)
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	pingCmd.Flags().StringP("access-token", "t", os.Getenv("REANA_ACCESS_TOKEN"), "Access token of the current user.")
}

func ping(token string, serverURL string) {
	respBytes := utils.NewRequest(token, serverURL, "/api/you")
	p := Ping{}

	if err := json.Unmarshal(respBytes, &p); err != nil {
		fmt.Printf("Could not unmarshal reponseBytes. %v", err)
	}
	response := fmt.Sprintf("REANA server: %s \n", serverURL) +
		fmt.Sprintf("REANA server version: %s \n", p.ServerVersion) +
		fmt.Sprintf("REANA client version: %s \n", version) +
		fmt.Sprintf("Authenticated as: <%s> \n", p.Email) +
		fmt.Sprintf("Status: %s ", "Connected")

	fmt.Println(response)
}
