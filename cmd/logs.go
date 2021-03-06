/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

// Pointers used for nullable values
type Logs struct {
	WorkflowLogs   *string               `json:"workflow_logs"`
	JobLogs        map[string]JobLogItem `json:"job_logs"`
	EngineSpecific *string               `json:"engine_specific"`
}

type JobLogItem struct {
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
compute_backend, docker_img, status and step.
`

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Get workflow logs.",
		Long:  logsDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString("access-token")
			if token == "" {
				token = os.Getenv("REANA_ACCESS_TOKEN")
			}
			serverURL := os.Getenv("REANA_SERVER_URL")
			workflow, _ := cmd.Flags().GetString("workflow")
			if workflow == "" {
				workflow = os.Getenv("REANA_WORKON")
			}

			if err := validation.ValidateAccessToken(token); err != nil {
				return err
			}
			if err := validation.ValidateServerURL(serverURL); err != nil {
				return err
			}
			if err := validation.ValidateWorkflow(workflow); err != nil {
				return err
			}
			if err := logs(cmd, token, workflow); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().Bool("json", false, "Get output in JSON format.")
	cmd.Flags().StringP("access-token", "t", "", "Access token of the current user.")
	cmd.Flags().
		StringP("workflow", "w", "", "Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.")
	cmd.Flags().StringSlice("filter", []string{}, logsFilterFlagDesc)
	cmd.Flags().Int64("page", 1, "Results page number (to be used with --size).")
	cmd.Flags().Int64("size", 0, "Size of results per page (to be used with --page).")

	return cmd
}

func logs(cmd *cobra.Command, token string, workflow string) error {
	filters, _ := cmd.Flags().GetStringSlice("filter")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	page, _ := cmd.Flags().GetInt64("page")
	size, _ := cmd.Flags().GetInt64("size")

	steps, chosenFilters, err := parseLogsFilters(filters)
	if err != nil {
		return err
	}

	logsParams := operations.NewGetWorkflowLogsParams()
	logsParams.SetAccessToken(&token)
	logsParams.SetWorkflowIDOrName(workflow)
	logsParams.SetPage(&page)
	logsParams.SetSteps(steps)
	if cmd.Flags().Changed("size") {
		logsParams.SetSize(&size)
	}

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	logsResp, err := api.Operations.GetWorkflowLogs(logsParams)
	if err != nil {
		return fmt.Errorf("workflow logs could not be retrieved:\n%v", err)
	}

	var workflowLogs Logs
	err = json.Unmarshal([]byte(logsResp.GetPayload().Logs), &workflowLogs)
	if err != nil {
		return err
	}

	err = filterJobLogs(&workflowLogs.JobLogs, chosenFilters)
	if err != nil {
		return err
	}

	if jsonOutput {
		err := utils.DisplayJsonOutput(workflowLogs, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		displayHumanFriendlyLogs(cmd, workflowLogs, steps)
	}

	return nil
}

func parseLogsFilters(filters []string) ([]string, map[string]string, error) {
	availableFilters := map[string]string{
		"step":            "job_name",
		"compute_backend": "compute_backend",
		"docker_img":      "docker_img",
		"status":          "status",
	}

	var steps []string
	chosenFilters := make(map[string]string)
	for _, filter := range filters {
		filterName, filterValue, err := utils.GetFilterNameAndValue(filter)
		if err != nil {
			return nil, nil, err
		}

		_, isFilterValid := availableFilters[filterName]
		if !isFilterValid {
			validFilters := make([]string, 0, len(availableFilters))
			for key := range availableFilters {
				validFilters = append(validFilters, key)
			}
			sort.Strings(validFilters)

			return nil, nil, fmt.Errorf(
				"filter '%s' is not valid\nAvailable filters are '%s'",
				filterName,
				strings.Join(validFilters, "' '"),
			)
		}

		if filterName == "step" {
			steps = append(steps, filterValue)
		} else {
			realValue, isKnownBackend := utils.ReanaComputeBackends[strings.ToLower(filterValue)]
			if filterName == "compute_backend" {
				if !isKnownBackend {
					return nil, nil, fmt.Errorf("compute_backend value %s is not valid", filterValue)
				}
				filterValue = realValue
			} else if filterName == "status" && !slices.Contains(utils.GetRunStatuses(true), filterValue) {
				return nil, nil, fmt.Errorf("input status value '%s' is not valid", filterValue)
			}
			realFilterName := availableFilters[filterName]
			chosenFilters[realFilterName] = filterValue
		}
	}

	return steps, chosenFilters, nil
}

func displayHumanFriendlyLogs(cmd *cobra.Command, logs Logs, steps []string) {
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

func displayOptionalItem(cmd *cobra.Command, item *string, title string, leadingMark string) {
	if item == nil || *item == "" {
		return
	}
	cmd.Printf("%s %s: %s\n", leadingMark, title, *item)
}

func filterJobLogs(jobLogs *map[string]JobLogItem, filters map[string]string) error {
	if len(filters) == 0 {
		return nil
	}

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
		for filterKey, filterValue := range filters {
			if jobLogValue[filterKey] != filterValue {
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
