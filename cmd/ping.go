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

const pingDesc = `
Check connection to REANA server.

The ` + "``ping``" + ` command allows to test connection to REANA server.
`

func newPingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check connection to REANA server.",
		Long:  pingDesc,
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
			if err := ping(token, serverURL); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")

	return cmd
}

func ping(token string, serverURL string) error {
	pingParams := operations.NewGetYouParams()
	pingParams.SetAccessToken(&token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	pingResp, err := api.Operations.GetYou(pingParams)
	if err != nil {
		return err
	}

	p := pingResp.Payload
	response := fmt.Sprintf("REANA server: %s \n", serverURL) +
		fmt.Sprintf("REANA server version: %s \n", p.ReanaServerVersion) +
		fmt.Sprintf("REANA client version: %s \n", version) +
		fmt.Sprintf("Authenticated as: <%s> \n", p.Email) +
		fmt.Sprintf("Status: %s ", "Connected")

	fmt.Println(response)
	return nil
}
