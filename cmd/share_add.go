/*
This file is part of REANA.
Copyright (C) 2023, 2024, 2025, 2026 CERN.

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
	"reanahub/reana-client-go/pkg/errorhandler"
	"reanahub/reana-client-go/pkg/validator"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const shareAddDesc = `Share a workflow with other users (read-only).

The ` + "`share-add`" + ` command allows sharing a workflow with other users.
The users will be able to view the workflow but not modify it.

Examples:

  $ reana-client share-add -w myanalysis.42 --user bob@cern.ch

  $ reana-client share-add -w myanalysis.42 --user bob@cern.ch
  --user cecile@cern.ch --message "Please review my analysis"
  --valid-until 2099-12-31
`

type shareAddOptions struct {
	token      string
	workflow   string
	users      []string
	message    string
	validUntil string
	jsonOutput bool
}

type shareAddResult struct {
	Workflow   string   `json:"workflow"`
	SharedWith []string `json:"shared_with"`
	Errors     []string `json:"errors"`
}

// newShareAddCmd creates a command to share a workflow with other users.
func newShareAddCmd() *cobra.Command {
	o := &shareAddOptions{}

	cmd := &cobra.Command{
		Use:   "share-add",
		Short: "Share a workflow with other users (read-only).",
		Long:  shareAddDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validator.ValidateAtLeastOne(
				cmd.Flags(), []string{"user"},
			); err != nil {
				return fmt.Errorf("%s\n%s", err.Error(), cmd.UsageString())
			}
			if err := validateShareExpiry(o.validUntil); err != nil {
				return err
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
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
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
		`Users to share the workflow with.`,
	)
	f.StringVarP(
		&o.message,
		"message",
		"m",
		"",
		`Optional message that is sent to the
	user(s) with the sharing invitation.`,
	)
	f.StringVar(
		&o.validUntil,
		"valid-until",
		"",
		`Optional date when access to the
	workflow will expire for the given
	user(s) (format: YYYY-MM-DD).`,
	)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "h", false, "Help for share-add")

	return cmd
}

func validateShareExpiry(value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return fmt.Errorf(
			"invalid value for 'valid-until': '%s' does not match the format 'YYYY-MM-DD'",
			value,
		)
	}
	return nil
}

func (o *shareAddOptions) run(cmd *cobra.Command) error {
	shareAddParams := operations.NewShareWorkflowParams()
	shareAddParams.SetWorkflowIDOrName(o.workflow)
	shareAddParams.SetShareDetails(operations.ShareWorkflowBody{
		Message:    o.message,
		ValidUntil: o.validUntil,
	})

	api, err := client.ApiClient(o.token)
	if err != nil {
		return err
	}

	shareErrors := []string{}
	sharedUsers := []string{}

	for _, user := range o.users {
		log.Infof("Sharing workflow %s with user %s", o.workflow, user)

		shareAddParams.ShareDetails.UserEmailToShareWith = &user
		_, err := api.Operations.ShareWorkflow(shareAddParams, nil)

		if err != nil {
			err := errorhandler.HandleApiError(err)
			shareErrors = append(
				shareErrors,
				fmt.Sprintf(
					"Failed to share %s with %s: %s",
					o.workflow,
					user,
					err.Error(),
				),
			)
		} else {
			sharedUsers = append(sharedUsers, user)
		}
	}

	if o.jsonOutput {
		if err := displayer.DisplayJsonOutput(
			shareAddResult{
				Workflow:   o.workflow,
				SharedWith: sharedUsers,
				Errors:     shareErrors,
			},
			cmd.OutOrStdout(),
		); err != nil {
			return err
		}
	} else {
		if len(sharedUsers) > 0 {
			displayer.DisplayMessage(
				fmt.Sprintf(
					"%s is now read-only shared with %s",
					o.workflow,
					strings.Join(sharedUsers, ", "),
				),
				displayer.Success,
				false,
				cmd.OutOrStdout(),
			)
		}
		for _, shareError := range shareErrors {
			displayer.DisplayMessage(
				shareError,
				displayer.Error,
				false,
				cmd.OutOrStdout(),
			)
		}
	}
	if len(shareErrors) > 0 {
		return config.ErrEmpty
	}

	return nil
}
