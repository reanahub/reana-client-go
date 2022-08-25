/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"

	"github.com/spf13/cobra"
)

const secretsListDesc = `
List user secrets.

Examples:

	$ reana-client secrets-list
`

type secretsListOptions struct {
	token string
}

// newSecretsListCmd creates a command to list user secrets.
func newSecretsListCmd() *cobra.Command {
	o := &secretsListOptions{}

	cmd := &cobra.Command{
		Use:   "secrets-list",
		Short: "List user secrets.",
		Long:  secretsListDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")

	return cmd
}

func (o *secretsListOptions) run(cmd *cobra.Command) error {
	listSecretsParams := operations.NewGetSecretsParams()
	listSecretsParams.SetAccessToken(&o.token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	listSecretsResp, err := api.Operations.GetSecrets(listSecretsParams)
	if err != nil {
		return err
	}

	header := []string{"name", "type"}
	var rows [][]string
	for _, secret := range listSecretsResp.Payload {
		row := []string{secret.Name, secret.Type}
		rows = append(rows, row)
	}
	displayer.DisplayTable(header, rows, cmd.OutOrStdout())

	return nil
}
