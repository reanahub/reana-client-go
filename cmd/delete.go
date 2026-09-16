/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"
	"fmt"

	"github.com/reanahub/reana-client-go/client/operations"
	"github.com/reanahub/reana-client-go/pkg/config"
	"github.com/reanahub/reana-client-go/pkg/displayer"
	"github.com/reanahub/reana-client-go/pkg/errorhandler"
	"github.com/reanahub/reana-client-go/pkg/workflows"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

const deleteDesc = `
Delete a workflow.

The ` + "``delete``" + ` command removes workflow run(s) from the database. Note that
the workspace and any open session attached to it will always be deleted,
even when ` + "``--include-workspace``" + ` is not specified. Note also that you can
remove all past runs of a workflow by specifying ` + "``--include-all-runs``" + ` flag.
Restarted runs share a workspace and must be deleted together by specifying
the ` + "``--include-all-restarts``" + ` flag.

Example:

$ reana-client delete -w myanalysis.42

$ reana-client delete -w myanalysis.42 --include-all-runs

$ reana-client delete -w myanalysis.42 --include-all-restarts
`

const (
	maxRestartLabels       = 10
	restartSeriesFirstPage = 1
	restartSeriesPageSize  = 1000
)

type deleteOptions struct {
	token              string
	workflow           string
	includeWorkspace   bool
	includeAllRuns     bool
	includeAllRestarts bool
}

type workflowDeletionFailure struct {
	label string
	err   error
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
	f.BoolVarP(
		&o.includeWorkspace,
		"include-workspace",
		"",
		true,
		"Delete workspace from REANA.",
	)
	f.BoolVarP(
		&o.includeAllRuns,
		"include-all-runs",
		"",
		false,
		"Delete all runs of a given workflow. Takes precedence over --include-all-restarts.",
	)
	f.BoolVar(
		&o.includeAllRestarts,
		"include-all-restarts",
		false,
		`Delete all restarted runs that share the same workspace as the selected run.
Without this flag, deletion fails for runs in a restart series.`,
	)

	return cmd
}

func (o *deleteOptions) run(cmd *cobra.Command) error {
	if o.includeAllRuns {
		return o.deleteAllRuns(cmd)
	}

	selectedRun, err := workflows.GetStatus(o.token, o.workflow)
	if err != nil {
		return err
	}
	if selectedRun == nil {
		return errors.New("workflow status response is empty")
	}

	fullName := selectedRun.Name
	if fullName == "" {
		fullName = o.workflow
	}
	if selectedRun.Status == "deleted" {
		displayer.DisplayMessage(
			fmt.Sprintf("Workflow run '%s' is already deleted.", fullName),
			displayer.Info,
			false,
			cmd.OutOrStdout(),
		)
		return nil
	}

	siblings, err := o.getRestartSiblings(selectedRun, fullName)
	if err != nil {
		return err
	}
	if len(siblings) == 0 {
		return o.deleteSingleRun(cmd, fullName)
	}

	seriesLabels := restartSeriesLabels(fullName, siblings)
	if !o.includeAllRestarts {
		return fmt.Errorf(
			"cannot delete workflow run '%s': it is part of a restart series. "+
				"Restarted runs share the same workspace, so deleting one run "+
				"would remove the shared workspace and leave other runs in an "+
				"inconsistent state.\nRelated runs: %s\nRerun the command with "+
				"--include-all-restarts to delete this run and its restarts",
			fullName,
			workflows.FormatRunLabelList(seriesLabels, maxRestartLabels),
		)
	}

	activeLabels := activeRestartLabels(siblings)
	if len(activeLabels) > 0 {
		return fmt.Errorf(
			"cannot delete workflow run '%s': some related restarts are still "+
				"active.\nActive runs: %s\nWait until they finish or stop them "+
				"first, then retry",
			fullName,
			workflows.FormatRunLabelList(activeLabels, maxRestartLabels),
		)
	}

	return o.deleteRestartSeries(cmd, selectedRun, fullName, siblings)
}

func (o *deleteOptions) deleteAllRuns(cmd *cobra.Command) error {
	if err := workflows.UpdateStatus(
		o.token,
		o.workflow,
		"deleted",
		o.includeWorkspace,
		true,
	); err != nil {
		return err
	}

	name, _ := workflows.GetNameAndRunNumber(o.workflow)
	displayer.DisplayMessage(
		fmt.Sprintf("All workflows named '%s' have been deleted", name),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)

	return nil
}

func (o *deleteOptions) deleteSingleRun(
	cmd *cobra.Command,
	fullName string,
) error {
	if err := workflows.UpdateStatus(
		o.token,
		fullName,
		"deleted",
		o.includeWorkspace,
		false,
	); err != nil {
		return err
	}

	message, err := workflows.StatusChangeMessage(fullName, "deleted")
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

func (o *deleteOptions) getRestartSiblings(
	selectedRun *operations.GetWorkflowStatusOKBody,
	fullName string,
) ([]*operations.GetWorkflowsOKBodyItemsItems0, error) {
	baseName, _, _ := workflows.ParseWorkflowRunNumber(fullName)
	majorKey := workflows.GetRunNumberMajorKey(fullName)
	if baseName == "" || majorKey == "" {
		return nil, nil
	}

	siblings := make([]*operations.GetWorkflowsOKBodyItemsItems0, 0)
	seenRuns := make(map[string]bool)
	fetchedRuns := int64(0)
	for page := int64(restartSeriesFirstPage); ; page++ {
		runs, totalRuns, err := workflows.ListRuns(
			o.token,
			baseName,
			config.GetRunStatuses(false),
			page,
			restartSeriesPageSize,
		)
		if err != nil {
			return nil, err
		}
		fetchedRuns += int64(len(runs))

		for _, run := range runs {
			if run == nil ||
				workflows.GetRunNumberMajorKey(run.Name) != majorKey ||
				isSelectedWorkflowRun(selectedRun, fullName, run) {
				continue
			}

			identity := workflowRunIdentity(run)
			if seenRuns[identity] {
				continue
			}
			seenRuns[identity] = true
			siblings = append(siblings, run)
		}

		if fetchedRuns == totalRuns {
			break
		}
		if len(runs) == 0 || fetchedRuns > totalRuns {
			return nil, fmt.Errorf(
				"cannot safely delete workflow run '%s': the restart-series "+
					"check received %d of %d workflow runs. Nothing was "+
					"deleted. Retry the command; if the problem persists, "+
					"contact your REANA administrator",
				fullName,
				fetchedRuns,
				totalRuns,
			)
		}
	}
	return siblings, nil
}

func (o *deleteOptions) deleteRestartSeries(
	cmd *cobra.Command,
	selectedRun *operations.GetWorkflowStatusOKBody,
	fullName string,
	siblings []*operations.GetWorkflowsOKBodyItemsItems0,
) error {
	deletedLabels := make([]string, 0, len(siblings)+1)
	failures := make([]workflowDeletionFailure, 0)

	selectedIDOrName := selectedRun.ID
	if selectedIDOrName == "" {
		selectedIDOrName = fullName
	}
	selectedLabel := workflows.FormatRunNumberLabel(fullName)
	if err := workflows.UpdateStatus(
		o.token,
		selectedIDOrName,
		"deleted",
		o.includeWorkspace,
		false,
	); err != nil {
		return restartSeriesDeletionError(
			fullName,
			nil,
			[]workflowDeletionFailure{{label: selectedLabel, err: err}},
		)
	}
	deletedLabels = append(deletedLabels, selectedLabel)

	for _, run := range siblings {
		idOrName := run.ID
		if idOrName == "" {
			idOrName = run.Name
		}
		label := workflows.FormatRunNumberLabel(run.Name)
		if err := workflows.UpdateStatus(
			o.token,
			idOrName,
			"deleted",
			false,
			false,
		); err != nil {
			failures = append(failures, workflowDeletionFailure{
				label: label,
				err:   err,
			})
		} else {
			deletedLabels = append(deletedLabels, label)
		}
	}

	if len(failures) > 0 {
		return restartSeriesDeletionError(fullName, deletedLabels, failures)
	}

	displayer.DisplayMessage(
		fmt.Sprintf(
			"Workflow run '%s' including its restarts has been deleted (%s).",
			fullName,
			workflows.FormatRunLabelList(
				restartSeriesLabels(fullName, siblings),
				maxRestartLabels,
			),
		),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)
	return nil
}

func restartSeriesDeletionError(
	fullName string,
	deletedLabels []string,
	failures []workflowDeletionFailure,
) error {
	failedLabels := make([]string, 0, len(failures))
	for _, failure := range failures {
		failedLabels = append(failedLabels, failure.label)
	}
	deletedRuns := workflows.FormatRunLabelList(
		deletedLabels,
		maxRestartLabels,
	)
	if deletedRuns == "" {
		deletedRuns = "none"
	}
	return fmt.Errorf(
		"workflow run '%s' could not be fully deleted.\nDeleted runs: %s\n"+
			"Failed runs: %s\nFirst error: %s\nRetry deleting the failed "+
			"run(s) with --include-all-restarts",
		fullName,
		deletedRuns,
		workflows.FormatRunLabelList(failedLabels, maxRestartLabels),
		errorhandler.HandleApiError(failures[0].err),
	)
}

func isSelectedWorkflowRun(
	selectedRun *operations.GetWorkflowStatusOKBody,
	fullName string,
	run *operations.GetWorkflowsOKBodyItemsItems0,
) bool {
	if selectedRun.ID != "" && run.ID != "" && selectedRun.ID == run.ID {
		return true
	}
	return fullName == run.Name
}

func workflowRunIdentity(run *operations.GetWorkflowsOKBodyItemsItems0) string {
	if run.ID != "" {
		return "id:" + run.ID
	}
	return "name:" + run.Name
}

func restartSeriesLabels(
	fullName string,
	siblings []*operations.GetWorkflowsOKBodyItemsItems0,
) []string {
	labels := make([]string, 0, len(siblings)+1)
	labels = append(labels, workflows.FormatRunNumberLabel(fullName))
	for _, run := range siblings {
		labels = append(labels, workflows.FormatRunNumberLabel(run.Name))
	}
	return labels
}

func activeRestartLabels(
	siblings []*operations.GetWorkflowsOKBodyItemsItems0,
) []string {
	labels := make([]string, 0)
	for _, run := range siblings {
		if slices.Contains(config.WorkflowProgressingStatuses, run.Status) {
			labels = append(labels, workflows.FormatRunNumberLabel(run.Name))
		}
	}
	return labels
}
