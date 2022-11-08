/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"

	"github.com/spf13/cobra"
)

const secretsDeleteDesc = `
Delete user secrets by name.

Examples:

	$ reana-client secrets-delete RUCIO_USERNAME
`

type secretsDeleteOptions struct {
	token   string
	secrets []string
}

// newSecretsDeleteCmd creates a command to delete user secrets by name.
func newSecretsDeleteCmd() *cobra.Command {
	o := &secretsDeleteOptions{}

	cmd := &cobra.Command{
		Use:   "secrets-delete",
		Short: "Delete user secrets by name.",
		Long:  secretsDeleteDesc,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.secrets = args
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")

	return cmd
}

func (o *secretsDeleteOptions) run(cmd *cobra.Command) error {
	deleteSecretsParams := operations.NewDeleteSecretsParams()
	deleteSecretsParams.SetAccessToken(&o.token)
	deleteSecretsParams.SetSecrets(o.secrets)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	deleteSecretsResp, err := api.Operations.DeleteSecrets(deleteSecretsParams)
	if err != nil {
		return handleSecretsDeleteApiError(err)
	}

	displayer.DisplayMessage(
		fmt.Sprintf(
			"Secrets %s were successfully deleted.",
			strings.Join(deleteSecretsResp.Payload, ", "),
		),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)

	return nil
}

// handleSecretsDeleteApiError handles the API error if the secrets were not found, displaying a human friendly message.
// Otherwise, return the same error.
func handleSecretsDeleteApiError(err error) error {
	notFoundErr, isNotFoundErr := err.(*operations.DeleteSecretsNotFound)
	if isNotFoundErr {
		return fmt.Errorf(
			"secrets %s do not exist. Nothing was deleted",
			strings.Join(notFoundErr.Payload, ", "),
		)
	}
	return err
}
