/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package workflows gives utility functions related to REANA's workflows.
package workflows

import (
	"fmt"
	"reanahub/reana-client-go/pkg/datautils"
	"strings"
	"time"
)

// GetNameAndRunNumber parses a string in the format 'name.number' and returns the workflow's name and number.
// Also works if only the workflow's name is provided.
func GetNameAndRunNumber(workflowName string) (string, string) {
	workflowNameAndRunNumber := strings.SplitN(workflowName, ".", 2)
	if len(workflowNameAndRunNumber) < 2 {
		return workflowName, ""
	}
	return workflowNameAndRunNumber[0], workflowNameAndRunNumber[1]
}

// GetDuration calculates and returns the duration the workflow, based on the given timestamps.
func GetDuration(runStartedAt, runFinishedAt *string) (any, error) {
	if runStartedAt == nil {
		return nil, nil
	}

	startTime, err := datautils.FromIsoToTimestamp(*runStartedAt)
	if err != nil {
		return nil, err
	}

	var endTime time.Time
	if runFinishedAt != nil {
		endTime, err = datautils.FromIsoToTimestamp(*runFinishedAt)
		if err != nil {
			return nil, err
		}
	} else {
		endTime = time.Now()
	}
	return endTime.Sub(startTime).Round(time.Second).Seconds(), nil
}

// StatusChangeMessage constructs the message to be displayed when a workflow changes its status.
func StatusChangeMessage(workflow, status string) (string, error) {
	var verb string
	if strings.HasSuffix(status, "ing") {
		verb = "is"
	} else if strings.HasSuffix(status, "ed") {
		verb = "has been"
	} else {
		return "", fmt.Errorf("unrecognised status %s", status)
	}

	return fmt.Sprintf("%s %s %s", workflow, verb, status), nil
}
