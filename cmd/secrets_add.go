/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/datautils"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/validator"
	"strings"

	"github.com/spf13/cobra"
)

const secretsAddDesc = `
Add secrets from literal string or from file.

Examples:

	$ reana-client secrets-add --env RUCIO_USERNAME=ruciouser

	$ reana-client secrets-add --file userkey.pem

	$ reana-client secrets-add --env VOMSPROXY_FILE=x509up_u1000

	            		   --file /tmp/x509up_u1000
`

type secretsAddOptions struct {
	token       string
	envSecrets  []string
	fileSecrets []string
	overwrite   bool
}

// newSecretsAddCmd creates a command to add secrets from literal string or from file.
func newSecretsAddCmd() *cobra.Command {
	o := &secretsAddOptions{}

	cmd := &cobra.Command{
		Use:   "secrets-add",
		Short: "Add secrets from literal string or from file.",
		Long:  secretsAddDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validator.ValidateAtLeastOne(
				cmd.Flags(), []string{"env", "file"},
			); err != nil {
				return fmt.Errorf("%s\n%s", err.Error(), cmd.UsageString())
			}
			for _, file := range o.fileSecrets {
				if err := validator.ValidateFile(file); err != nil {
					return fmt.Errorf("invalid value for '--file': %s", err.Error())
				}
			}
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringSliceVar(&o.envSecrets, "env", []string{}, `Secrets to be uploaded from literal string.
e.g. PASSWORD=password123`)
	f.StringSliceVar(&o.fileSecrets, "file", []string{}, "Secrets to be uploaded from file.")
	f.BoolVar(&o.overwrite, "overwrite", false, "Overwrite the secret if already present.")

	return cmd
}

func (o *secretsAddOptions) run(cmd *cobra.Command) error {
	secrets, secretNames, err := parseSecrets(o.envSecrets, o.fileSecrets)
	if err != nil {
		return err
	}

	addSecretsParams := operations.NewAddSecretsParams()
	addSecretsParams.SetAccessToken(&o.token)
	addSecretsParams.SetOverwrite(&o.overwrite)
	addSecretsParams.SetSecrets(secrets)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	_, err = api.Operations.AddSecrets(addSecretsParams)
	if err != nil {
		return err
	}

	displayer.DisplayMessage(
		fmt.Sprintf("Secrets %s were successfully uploaded.", strings.Join(secretNames, ", ")),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)

	return nil
}

// parseSecrets Parses env and file secrets into a map of secrets to be sent to the server and a slice of their names.
func parseSecrets(
	envSecrets []string,
	fileSecrets []string,
) (map[string]operations.AddSecretsParamsBodyAnon, []string, error) {
	secrets := make(map[string]operations.AddSecretsParamsBodyAnon)
	var secretNames []string

	for _, envLiteral := range envSecrets {
		key, value, err := datautils.SplitKeyValue(envLiteral)
		if err != nil {
			return nil, nil, fmt.Errorf(
				`option "%s" is invalid:
for literal strings use "SECRET_NAME=VALUE" format`,
				envLiteral,
			)
		}
		encodedValue := base64.StdEncoding.EncodeToString([]byte(value))
		secretNames = append(secretNames, key)
		secrets[key] = operations.AddSecretsParamsBodyAnon{
			Type:  "env",
			Value: encodedValue,
		}
	}

	for _, filePath := range fileSecrets {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"file %s could not be uploaded: %s",
				filePath, err.Error(),
			)
		}
		encodedData := base64.StdEncoding.EncodeToString(data)
		fileName := filepath.Base(filePath)
		secretNames = append(secretNames, fileName)
		secrets[fileName] = operations.AddSecretsParamsBodyAnon{
			Type:  "file",
			Value: encodedData,
		}
	}

	return secrets, secretNames, nil
}
