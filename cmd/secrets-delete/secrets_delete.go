/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package secrets_delete provides the command to delete user secrets by name.
package secrets_delete

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"

	"github.com/spf13/cobra"
)

const description = `
Delete user secrets by name.

Examples:

	$ reana-client secrets-delete PASSWORD
`

type options struct {
	token   string
	secrets []string
}

// NewCmd creates a command to delete user secrets by name.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "secrets-delete",
		Short: "Delete user secrets by name.",
		Long:  description,
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

func (o *options) run(cmd *cobra.Command) error {
	deleteSecretsParams := operations.NewDeleteSecretsParams()
	deleteSecretsParams.SetAccessToken(&o.token)
	deleteSecretsParams.SetSecrets(o.secrets)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	deleteSecretsResp, err := api.Operations.DeleteSecrets(deleteSecretsParams)
	if err != nil {
		return handleDeleteSecretsError(err)
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

// handleDeleteSecretsError handles the API error if the secrets were not found, displaying a human friendly message.
// Otherwise, return the same error.
func handleDeleteSecretsError(err error) error {
	notFoundErr, isNotFoundErr := err.(*operations.DeleteSecretsNotFound)
	if isNotFoundErr {
		return fmt.Errorf(
			"secrets ['%s'] do not exist. Nothing was deleted",
			strings.Join(notFoundErr.Payload, "', '"),
		)
	}
	return err
}
