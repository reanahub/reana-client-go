/*
This file is part of REANA.
Copyright (C) 2022, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"reanahub/reana-client-go/pkg/errorhandler"

	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const pingDesc = `
Check connection to REANA server.

The ` + "``ping``" + ` command allows to test connection to REANA server.
`

type pingOptions struct {
	token     string
	serverURL string
}

// newPingCmd creates a command to ping the REANA server.
func newPingCmd() *cobra.Command {
	o := &pingOptions{}

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check connection to REANA server.",
		Long:  pingDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(
		&o.token,
		"access-token",
		"t",
		"",
		"Access token of the current user.",
	)

	return cmd
}

func (o *pingOptions) run(cmd *cobra.Command) error {
	pingParams := operations.NewGetYouParams()

	api, err := client.ApiClient(o.token)
	if err != nil {
		return err
	}
	pingResp, err := api.Operations.GetYou(pingParams, nil)
	if err != nil {
		var transportError *url.Error
		if errors.As(err, &transportError) {
			return auth.ConnectionError(o.serverURL, o.serverURL, err)
		}
		return auth.ForServer(
			o.serverURL,
			"saved login",
			errorhandler.HandleApiError(err),
		)
	}

	status, err := client.TLSStatus()
	if err != nil {
		return err
	}
	p := pingResp.Payload
	response := fmt.Sprintf("REANA server: %s \n", o.serverURL) +
		fmt.Sprintf("REANA server version: %s \n", p.ReanaServerVersion) +
		fmt.Sprintf("REANA client version: %s \n", version) +
		fmt.Sprintf("Authenticated as: <%s> \n", p.Email) +
		fmt.Sprintf("Status: %s\nTLS verification: %s", "Connected", status)

	cmd.Println(response)

	return nil
}
