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

	"github.com/spf13/cobra"
)

const deleteDesc = `
Delete a workflow.

The ` + "``delete``" + ` command removes workflow run(s) from the database. Note that
the workspace will always be deleted, even when ` + "``--include-workspace``" + ` is
not specified. Note also that you can remove all past runs of a workflow by
specifying ` + "``--include-all-runs``" + ` flag.

Example:

$ reana-client delete -w myanalysis.42

$ reana-client delete -w myanalysis.42 --include-all-runs
`

type deleteOptions struct {
	token            string
	workflow         string
	includeWorkspace bool
	includeAllRuns   bool
}

// newDeleteCmd creates a command to delete a workflow.
func newDeleteCmd() *cobra.Command {
	o := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a workflow.",
		Long:  deleteDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
	f.BoolVarP(&o.includeWorkspace, "include-workspace", "", true, "Delete workspace from REANA.")
	f.BoolVarP(
		&o.includeAllRuns,
		"include-all-runs",
		"",
		false,
		"Delete all runs of a given workflow.",
	)

	return cmd
}

func (o *deleteOptions) run(cmd *cobra.Command) error {
	err := workflows.UpdateStatus(
		o.token,
		o.workflow,
		"deleted",
		o.includeWorkspace,
		o.includeAllRuns,
	)
	if err != nil {
		return err
	}

	var message string
	if o.includeAllRuns {
		name, _ := workflows.GetNameAndRunNumber(o.workflow)
		message = fmt.Sprintf("All workflows named '%s' have been deleted", name)
	} else {
		message, err = workflows.StatusChangeMessage(o.workflow, "deleted")
		if err != nil {
			return err
		}
	}
	displayer.DisplayMessage(message, displayer.Success, false, cmd.OutOrStdout())

	return nil
}
