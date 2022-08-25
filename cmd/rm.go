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
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"

	"github.com/spf13/cobra"
)

const rmDesc = `
Delete files from workspace.

The ` + "``rm``" + ` command allow to delete files and directories from workspace.
Note that you can use glob to remove similar files.

Examples:

	$ reana-client rm -w myanalysis.42 data/mydata.csv

	$ reana-client rm -w myanalysis.42 'data/*root*'
`

type rmOptions struct {
	token     string
	workflow  string
	fileNames []string
}

// newRmCmd creates a command to delete files from workspace.
func newRmCmd() *cobra.Command {
	o := &rmOptions{}

	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Delete files from workspace.",
		Long:  rmDesc,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.fileNames = args
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w", "",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)

	return cmd
}

func (o *rmOptions) run(cmd *cobra.Command) error {
	api, err := client.ApiClient()
	if err != nil {
		return err
	}

	hasError := false
	for _, fileName := range o.fileNames {
		rmParams := operations.NewDeleteFileParams()
		rmParams.SetAccessToken(&o.token)
		rmParams.SetWorkflowIDOrName(o.workflow)
		rmParams.SetFileName(fileName)

		rmResp, err := api.Operations.DeleteFile(rmParams)
		if err != nil {
			return err
		}

		deleted := rmResp.Payload.Deleted
		failed := rmResp.Payload.Failed
		if len(deleted) == 0 && len(failed) == 0 {
			hasError = true
			displayer.DisplayMessage(
				fmt.Sprintf("%s did not match any existing file", fileName),
				displayer.Error,
				false,
				cmd.OutOrStdout(),
			)
		}

		var freedSpace int64
		for file, fileInfo := range deleted {
			freedSpace += fileInfo.Size
			displayer.DisplayMessage(
				fmt.Sprintf("File %s was successfully deleted.", file),
				displayer.Success,
				false,
				cmd.OutOrStdout(),
			)
		}
		for file, errorInfo := range failed {
			hasError = true
			displayer.DisplayMessage(
				fmt.Sprintf("Something went wrong while deleting %s.\n%s", file, errorInfo.Error),
				displayer.Error,
				false,
				cmd.OutOrStdout(),
			)
		}
		if freedSpace > 0 {
			displayer.DisplayMessage(
				fmt.Sprintf("%d bytes freed up.", freedSpace),
				displayer.Success,
				false,
				cmd.OutOrStdout(),
			)
		}
	}
	if hasError {
		return config.EmptyError
	}
	return nil
}
