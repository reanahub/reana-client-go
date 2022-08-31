/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package ping provides the command to ping the REANA server.
package ping

import (
	"fmt"
	"reanahub/reana-client-go/pkg/config"

	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const description = `
Check connection to REANA server.

The ` + "``ping``" + ` command allows to test connection to REANA server.
`

type options struct {
	token     string
	serverURL string
}

// NewCmd creates a command to ping the REANA server.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check connection to REANA server.",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")

	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
	pingParams := operations.NewGetYouParams()
	pingParams.SetAccessToken(&o.token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	pingResp, err := api.Operations.GetYou(pingParams)
	if err != nil {
		return err
	}

	p := pingResp.Payload
	response := fmt.Sprintf("REANA server: %s \n", o.serverURL) +
		fmt.Sprintf("REANA server version: %s \n", p.ReanaServerVersion) +
		fmt.Sprintf("REANA client version: %s \n", config.Version) +
		fmt.Sprintf("Authenticated as: <%s> \n", p.Email) +
		fmt.Sprintf("Status: %s ", "Connected")

	cmd.Println(response)

	return nil
}
