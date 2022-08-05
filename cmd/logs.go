/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

const logsDesc = `
Get workflow logs.

The ` + "``logs``" + ` command allows to retrieve logs of running workflow. Note that
only finished steps of the workflow are returned, the logs of the currently
processed step is not returned until it is finished.

Examples:

$ reana-client logs -w myanalysis.42

$ reana-client logs -w myanalysis.42 -s 1st_ste
`

const logsFilterFlagDesc = `Filter job logs to include only those steps that
match certain filtering criteria. Use --filter
name=value pairs. Available filters are
compute_backend, docker_img, status and step.`

// logs struct that contains the logs of a workflow.
// Pointers used for nullable values
type logs struct {
	WorkflowLogs   *string               `json:"workflow_logs"`
	JobLogs        map[string]jobLogItem `json:"job_logs"`
	EngineSpecific *string               `json:"engine_specific"`
}

// jobLogItem struct that contains the log information of a job.
type jobLogItem struct {
	WorkflowUuid   string  `json:"workflow_uuid"`
	JobName        string  `json:"job_name"`
	ComputeBackend string  `json:"compute_backend"`
	BackendJobId   string  `json:"backend_job_id"`
	DockerImg      string  `json:"docker_img"`
	Cmd            string  `json:"cmd"`
	Status         string  `json:"status"`
	Logs           string  `json:"logs"`
	StartedAt      *string `json:"started_at"`
	FinishedAt     *string `json:"finished_at"`
}

type logsOptions struct {
	token      string
	workflow   string
	jsonOutput bool
	filters    []string
	page       int64
	size       int64
}

// newLogsCmd creates a command to get workflow logs.
func newLogsCmd() *cobra.Command {
	o := &logsOptions{}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Get workflow logs.",
		Long:  logsDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w",
		"",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	f.BoolVar(&o.jsonOutput, "json", false, "Get output in JSON format.")
	f.StringSliceVar(&o.filters, "filter", []string{}, logsFilterFlagDesc)
	f.Int64Var(&o.page, "page", 1, "Results page number (to be used with --size).")
	f.Int64Var(&o.size, "size", 0, "Size of results per page (to be used with --page).")

	return cmd
}

func (o *logsOptions) run(cmd *cobra.Command) error {
	filters, err := parseLogsFilters(o.filters)
	if err != nil {
		return err
	}
	steps, err := filters.GetMulti("step")
	if err != nil {
		return err
	}

	logsParams := operations.NewGetWorkflowLogsParams()
	logsParams.SetAccessToken(&o.token)
	logsParams.SetWorkflowIDOrName(o.workflow)
	logsParams.SetPage(&o.page)
	logsParams.SetSteps(steps)
	if cmd.Flags().Changed("size") {
		logsParams.SetSize(&o.size)
	}

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	logsResp, err := api.Operations.GetWorkflowLogs(logsParams)
	if err != nil {
		return err
	}

	var workflowLogs logs
	err = json.Unmarshal([]byte(logsResp.GetPayload().Logs), &workflowLogs)
	if err != nil {
		return err
	}

	err = filterJobLogs(&workflowLogs.JobLogs, filters)
	if err != nil {
		return err
	}

	if o.jsonOutput {
		err := utils.DisplayJsonOutput(workflowLogs, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		displayHumanFriendlyLogs(cmd, workflowLogs, steps)
	}

	return nil
}

// parseLogsFilters parses a list of filters in the format 'filter=value', for the 'logs' command.
// The steps filters are returned as a slice of strings, while the other filters are returned as a map.
// Returns an error if any of the given filters are not valid.
func parseLogsFilters(filterInput []string) (utils.Filters, error) {
	singleValueFilters := []string{"compute_backend", "docker_img", "status"}
	multiValueFilters := []string{"step"}

	filters, err := utils.NewFilters(singleValueFilters, multiValueFilters, filterInput)
	if err != nil {
		return filters, err
	}

	err = filters.ValidateValues("status", utils.GetRunStatuses(true))
	if err != nil {
		return filters, err
	}

	err = filters.ValidateValues("compute_backend", utils.ReanaComputeBackendKeys)
	if err != nil {
		return filters, err
	}

	return filters, nil
}

// filterJobLogs returns a subset of jobLogs based on the given filters.
func filterJobLogs(jobLogs *map[string]jobLogItem, filters utils.Filters) error {
	// Convert to a map based on json properties
	var jobLogsMap map[string]map[string]string
	jsonLogs, err := json.Marshal(jobLogs)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonLogs, &jobLogsMap)
	if err != nil {
		return err
	}

	var unwantedLogs []string
	for jobLogKey, jobLogValue := range jobLogsMap {
		for _, filterKey := range filters.SingleFilterKeys {
			filterValue, _ := filters.GetSingle(filterKey)
			if filterKey == "compute_backend" {
				filterValue = utils.ReanaComputeBackends[strings.ToLower(filterValue)]
			}
			if filterValue != "" && jobLogValue[filterKey] != filterValue {
				unwantedLogs = append(unwantedLogs, jobLogKey)
				break
			}
		}
	}

	for _, log := range unwantedLogs {
		delete(*jobLogs, log)
	}
	return nil
}

// displayHumanFriendlyLogs displays the logs in a human friendly way.
func displayHumanFriendlyLogs(cmd *cobra.Command, logs logs, steps []string) {
	leadingMark := "==>"

	if logs.WorkflowLogs != nil && *logs.WorkflowLogs != "" {
		cmd.Printf("%s Workflow engine logs\n", leadingMark)
		cmd.Println(*logs.WorkflowLogs)
	}

	if logs.EngineSpecific != nil && *logs.EngineSpecific != "" {
		cmd.Printf("\n%s Engine internal logs\n", leadingMark)
		cmd.Println(*logs.EngineSpecific)
	}

	if len(steps) > 0 {
		var returnedStepNames, missingStepNames []string
		for _, jobItem := range logs.JobLogs {
			returnedStepNames = append(returnedStepNames, jobItem.JobName)
		}

		for _, step := range steps {
			if !slices.Contains(returnedStepNames, step) {
				missingStepNames = append(missingStepNames, step)
			}
		}

		if len(missingStepNames) > 0 {
			cmd.PrintErrf(
				"The logs of step(s) %s were not found, check for spelling mistakes in the step names\n",
				strings.Join(missingStepNames, ","),
			)
		}
	}

	if len(logs.JobLogs) > 0 {
		cmd.Printf("\n%s Job logs\n", leadingMark)
		for jobId, jobItem := range logs.JobLogs {
			jobNameOrId := jobId
			if jobItem.JobName != "" {
				jobNameOrId = jobItem.JobName
			}
			cmd.Printf("%s Step: %s\n", leadingMark, jobNameOrId)

			displayOptionalItem(cmd, &jobItem.WorkflowUuid, "Workflow ID", leadingMark)
			displayOptionalItem(cmd, &jobItem.ComputeBackend, "Compute backend", leadingMark)
			displayOptionalItem(cmd, &jobItem.BackendJobId, "Job ID", leadingMark)
			displayOptionalItem(cmd, &jobItem.DockerImg, "Docker image", leadingMark)
			displayOptionalItem(cmd, &jobItem.Cmd, "Command", leadingMark)
			displayOptionalItem(cmd, &jobItem.Status, "Status", leadingMark)
			displayOptionalItem(cmd, jobItem.StartedAt, "Started", leadingMark)
			displayOptionalItem(cmd, jobItem.FinishedAt, "Finished", leadingMark)

			if jobItem.Logs != "" {
				cmd.Printf("%s Logs:\n", leadingMark)
				cmd.Println(jobItem.Logs)
			} else {
				cmd.Printf("Step %s emitted no logs.", jobNameOrId)
			}
		}
	}
}

// displayOptionalItem displays an optional item if it is not nil or an empty string.
func displayOptionalItem(cmd *cobra.Command, item *string, title string, leadingMark string) {
	if item == nil || *item == "" {
		return
	}
	cmd.Printf("%s %s: %s\n", leadingMark, title, *item)
}
