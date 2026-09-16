/*
This file is part of REANA.
Copyright (C) 2023, 2024, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"strings"

	"github.com/reanahub/reana-client-go/client"
	"github.com/reanahub/reana-client-go/client/operations"
	"github.com/reanahub/reana-client-go/pkg/config"
	"github.com/reanahub/reana-client-go/pkg/displayer"
	"github.com/reanahub/reana-client-go/pkg/errorhandler"
	"github.com/reanahub/reana-client-go/pkg/validator"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const shareRemoveDesc = `Unshare a workflow.

The ` + "`share-remove`" + ` command allows for unsharing a workflow. The workflow
will no longer be visible to the users with whom it was shared.

Example:

  $ reana-client share-remove -w myanalysis.42 --user bob@example.org
`

type shareRemoveOptions struct {
	token      string
	workflow   string
	users      []string
	jsonOutput bool
}

type shareRemoveResult struct {
	Workflow     string   `json:"workflow"`
	UnsharedWith []string `json:"unshared_with"`
	Errors       []string `json:"errors"`
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
			if err := validator.ValidateAtLeastOne(
				cmd.Flags(), []string{"user"},
			); err != nil {
				return fmt.Errorf("%s\n%s", err.Error(), cmd.UsageString())
			}
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
	f.StringVarP(
		&o.token,
		"access-token",
		"t",
		"",
		"Access token of the current user.",
	)
	f.StringSliceVarP(
		&o.users,
		"user",
		"u",
		[]string{},
		`Users to unshare the workflow with.`,
	)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "h", false, "Help for share-remove")

	return cmd
}

func (o *shareRemoveOptions) run(cmd *cobra.Command) error {
	shareRemoveParams := operations.NewUnshareWorkflowParams()
	shareRemoveParams.SetWorkflowIDOrName(o.workflow)

	api, err := client.ApiClient(o.token)
	if err != nil {
		return err
	}

	unshareErrors := []string{}
	unsharedUsers := []string{}

	for _, user := range o.users {
		log.Infof("Unsharing workflow %s with user %s", o.workflow, user)

		shareRemoveParams.SetUserEmailToUnshareWith(user)
		_, err := api.Operations.UnshareWorkflow(shareRemoveParams, nil)

		if err != nil {
			err := errorhandler.HandleApiError(err)
			unshareErrors = append(
				unshareErrors,
				fmt.Sprintf(
					"Failed to unshare %s with %s: %s",
					o.workflow,
					user,
					err.Error(),
				),
			)
		} else {
			unsharedUsers = append(unsharedUsers, user)
		}
	}

	if o.jsonOutput {
		if err := displayer.DisplayJsonOutput(
			shareRemoveResult{
				Workflow:     o.workflow,
				UnsharedWith: unsharedUsers,
				Errors:       unshareErrors,
			},
			cmd.OutOrStdout(),
		); err != nil {
			return err
		}
	} else {
		if len(unsharedUsers) > 0 {
			displayer.DisplayMessage(
				fmt.Sprintf(
					"%s is no longer shared with %s",
					o.workflow,
					strings.Join(unsharedUsers, ", "),
				),
				displayer.Success,
				false,
				cmd.OutOrStdout(),
			)
		}
		for _, unshareError := range unshareErrors {
			displayer.DisplayMessage(
				unshareError,
				displayer.Error,
				false,
				cmd.OutOrStdout(),
			)
		}
	}
	if len(unshareErrors) > 0 {
		return config.ErrEmpty
	}

	return nil
}
