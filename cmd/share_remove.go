/*
This file is part of REANA.
Copyright (C) 2023 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/errorhandler"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const shareRemoveDesc = `Unshare a workflow.

The ` + `share-remove` + ` command allows for unsharing a workflow. The workflow
will no longer be visible to the users with whom it was shared.

Example:

  $ reana-client share-remove -w myanalysis.42 --user bob@example.org
`

type shareRemoveOptions struct {
	token    string
	workflow string
	users    []string
}

// newShareRemoveCmd creates a command to unshare a workflow.
func newShareRemoveCmd() *cobra.Command {
	o := &shareRemoveOptions{}

	cmd := &cobra.Command{
		Use:   "share-remove",
		Short: "Unshare a workflow.",
		Long:  shareRemoveDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w",
		"",
		`Name or UUID of the workflow. Overrides value of 
	REANA_WORKON environment variable.`,
	)
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringSliceVarP(&o.users, "user", "u", []string{}, `Users to unshare the workflow with.`)
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "h", false, "Help for share-remove")

	return cmd
}

func (o *shareRemoveOptions) run(cmd *cobra.Command) error {
	shareRemoveParams := operations.NewUnshareWorkflowParams()
	shareRemoveParams.SetAccessToken(&o.token)
	shareRemoveParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}

	shareErrors := []string{}
	sharedUsers := []string{}

	for _, user := range o.users {
		log.Infof("Unsharing workflow %s with user %s", o.workflow, user)

		shareRemoveParams.SetUserEmailToUnshareWith(user)
		_, err := api.Operations.UnshareWorkflow(shareRemoveParams)

		if err != nil {
			err := errorhandler.HandleApiError(err)
			shareErrors = append(
				shareErrors,
				fmt.Sprintf("Failed to unshare %s with %s: %s", o.workflow, user, err.Error()),
			)
		} else {
			sharedUsers = append(sharedUsers, user)
		}
	}

	if len(sharedUsers) > 0 {
		displayer.DisplayMessage(
			fmt.Sprintf(
				"%s is no longer shared with %s",
				o.workflow,
				strings.Join(sharedUsers, ", "),
			),
			displayer.Success,
			false,
			cmd.OutOrStdout(),
		)
	}
	if len(shareErrors) > 0 {
		for _, err := range shareErrors {
			displayer.DisplayMessage(err, displayer.Error, false, cmd.OutOrStdout())
		}
	}

	return nil
}
