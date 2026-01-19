/*
This file is part of REANA.
Copyright (C) 2022, 2024, 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package config gives constants and small functions that specify the REANA client configuration.
package config

import "errors"

// FilesBlacklist list of files to be ignored.
var FilesBlacklist = []string{".git/", "/.git/"}

// InteractiveSessionTypes list of supported types of interactive sessions.
var InteractiveSessionTypes = []string{"jupyter"}

// ReanaComputeBackends maps the backends' command line references to their real names.
var ReanaComputeBackends = map[string]string{
	"kubernetes": "Kubernetes",
	"htcondor":   "HTCondor",
	"slurm":      "Slurm",
}

// ReanaComputeBackendKeys valid options for compute backends, used in the command line.
// These keys are the same used in ReanaComputeBackends.
var ReanaComputeBackendKeys = []string{"kubernetes", "htcondor", "slurm"}

// LeadingMark prefix used when displaying headers or important messages.
var LeadingMark = "==>"

var WorkflowCompletedStatuses = []string{"finished", "failed", "stopped"}

var WorkflowProgressingStatuses = []string{
	"created",
	"running",
	"queued",
	"pending",
}

// GetRunStatuses provides a list of currently supported run statuses.
// Includes the deleted status if includeDeleted is set to true.
func GetRunStatuses(includeDeleted bool) []string {
	runStatuses := append(
		WorkflowCompletedStatuses,
		WorkflowProgressingStatuses...)

	if includeDeleted {
		runStatuses = append(runStatuses, "deleted")
	}
	return runStatuses
}

// UpdateStatusActions provides a list of supported actions for updating the workflow status.
var UpdateStatusActions = []string{"start", "stop", "deleted"}

// DuMultiFilters available filters with multiple values in du command.
var DuMultiFilters = []string{"size", "name"}

// ListMultiFilters available filters with multiple values in list command.
var ListMultiFilters = []string{"name", "status"}

// LogsSingleFilters available filters with a single value in logs command.
var LogsSingleFilters = []string{"compute_backend", "docker_img", "status"}

// LogsMultiFilters available filters with multiple values in logs command.
var LogsMultiFilters = []string{"step"}

// QuotaReports available reports in quota-show command.
var QuotaReports = []string{"limit", "usage"}

// AvailableOperationalOptions available operational options and respective translations according to the workflow type.
var AvailableOperationalOptions = map[string]map[string]string{
	"CACHE":          {"serial": "CACHE"},
	"FROM":           {"serial": "FROM"},
	"TARGET":         {"serial": "TARGET", "cwl": "--target"},
	"toplevel":       {"yadage": "toplevel"},
	"initdir":        {"yadage": "initdir"},
	"initfiles":      {"yadage": "initfiles"},
	"accept_metadir": {"yadage": "accept_metadir"},
	"report":         {"snakemake": "report"},
}

// CheckInterval interval between workflow status check, in seconds.
var CheckInterval = 5

// ErrEmpty Error to be used in case we want to return an error that isn't displayed to the user.
// Useful when the command already prints the errors occurred.
var ErrEmpty = errors.New("")

// StdoutChar used to refer to the standard output.
var StdoutChar = "-"
