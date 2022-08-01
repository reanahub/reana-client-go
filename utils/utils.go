/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package utils provides utility functions and constants to be used across the rest of the program.
package utils

import (
	"bytes"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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

// GetRunStatuses provides a list of currently supported run statuses.
// Includes the deleted status if includeDeleted is set to true.
func GetRunStatuses(includeDeleted bool) []string {
	runStatuses := []string{
		"created",
		"running",
		"finished",
		"failed",
		"stopped",
		"queued",
		"pending",
	}
	if includeDeleted {
		runStatuses = append(runStatuses, "deleted")
	}
	return runStatuses
}

// ExecuteCommand executes a cobra command with the given args.
// Returns the output of the command and any error it may provide.
func ExecuteCommand(cmd *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.String(), err
}

// HasAnyPrefix checks if the string s has any prefixes, by running strings.HasPrefix for each one.
func HasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// FromIsoToTimestamp converts a string date in the ISO format to a timestamp.
func FromIsoToTimestamp(date string) (time.Time, error) {
	timestamp, err := time.Parse("2006-01-02T15:04:05", date)
	if err != nil {
		return time.Time{}, err
	}
	return timestamp, nil
}

// GetWorkflowNameAndRunNumber parses a string in the format 'name.number' and returns the workflow's name and number.
// Also works if only the workflow's name is provided.
func GetWorkflowNameAndRunNumber(workflowName string) (string, string) {
	workflowNameAndRunNumber := strings.SplitN(workflowName, ".", 2)
	if len(workflowNameAndRunNumber) < 2 {
		return workflowName, ""
	}
	return workflowNameAndRunNumber[0], workflowNameAndRunNumber[1]
}

// FormatSessionURI takes the serverURL, its token and a path, and formats them into a session URI.
func FormatSessionURI(serverURL string, path string, token string) string {
	return serverURL + path + "?token=" + token
}

// LogCmdFlags logs all the flags set in the given command.
func LogCmdFlags(cmd *cobra.Command) {
	log.Debugf("command: %s", cmd.CalledAs())
	cmd.Flags().Visit(func(f *pflag.Flag) {
		log.Debugf("%s: %s", f.Name, f.Value)
	})
}
