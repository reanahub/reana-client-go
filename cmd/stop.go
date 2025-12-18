/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/workflows"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const stopDesc = `
Stop a running workflow.

The ` + "``stop``" + ` command allows to hard-stop the running workflow process. Note
that soft-stopping of the workflow is currently not supported. This command
should be therefore used with care, only if you are absolutely sure that
there is no point in continuing the running the workflow.

Example:

  $ reana-client stop -w myanalysis.42 --force
`

type stopOptions struct {
	token    string
	workflow string
	force    bool
}

// newStopCmd creates a command to stop a running workflow.
func newStopCmd() *cobra.Command {
	o := &stopOptions{}

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running workflow.",
		Long:  stopDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w",
		"",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	f.BoolVar(
		&o.force,
		"force",
		false,
		"Stop a workflow without waiting for jobs to finish.",
	)

	return cmd
}

func (o *stopOptions) run(cmd *cobra.Command) error {
	if !o.force {
		return fmt.Errorf(
			"graceful stop not implemented yet. If you really want to stop your " +
				"workflow without waiting for jobs to finish use: --force option",
		)
	}

	log.Infof("Sending a request to stop workflow %s", o.workflow)
	err := workflows.UpdateStatus(
		o.token,
		o.workflow,
		"stop",
		false,
		false,
	)
	if err != nil {
		return err
	}

	message, err := workflows.StatusChangeMessage(o.workflow, "stopped")
	if err != nil {
		return err
	}

	displayer.DisplayMessage(
		message,
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)
	return nil
}
