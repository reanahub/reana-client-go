/*
This file is part of REANA.
Copyright (C) 2022, 2024, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package workflows gives utility functions related to REANA's workflows.
package workflows

import (
	"fmt"
	"strings"
	"time"

	"github.com/reanahub/reana-client-go/pkg/datautils"
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

// ParseWorkflowRunNumber splits a workflow run name into its base name and
// major and minor run-number components.
func ParseWorkflowRunNumber(
	fullName string,
) (baseName, major, minor string) {
	if fullName == "" {
		return "", "", ""
	}

	parts := strings.Split(fullName, ".")
	firstNumericPart := len(parts)
	for firstNumericPart > 0 && isNumeric(parts[firstNumericPart-1]) {
		firstNumericPart--
	}

	baseName = strings.Join(parts[:firstNumericPart], ".")
	numericParts := parts[firstNumericPart:]
	if len(numericParts) == 0 {
		return baseName, "", ""
	}

	major = numericParts[0]
	minor = strings.Join(numericParts[1:], ".")
	return baseName, major, minor
}

// GetRunNumberMajorKey returns the common key shared by restarted workflow
// runs, consisting of the base name and major run number.
func GetRunNumberMajorKey(fullName string) string {
	baseName, major, _ := ParseWorkflowRunNumber(fullName)
	if baseName == "" || major == "" {
		return ""
	}
	return baseName + "." + major
}

// FormatRunNumberLabel returns a compact user-facing workflow run label.
func FormatRunNumberLabel(fullName string) string {
	_, major, minor := ParseWorkflowRunNumber(fullName)
	if major == "" {
		return fullName
	}
	if minor == "" {
		return "#" + major
	}
	return "#" + major + "." + minor
}

// FormatRunLabelList joins workflow run labels and truncates long lists.
func FormatRunLabelList(labels []string, maxLabels int) string {
	nonEmptyLabels := make([]string, 0, len(labels))
	for _, label := range labels {
		if label != "" {
			nonEmptyLabels = append(nonEmptyLabels, label)
		}
	}

	if maxLabels <= 0 || len(nonEmptyLabels) <= maxLabels {
		return strings.Join(nonEmptyLabels, ", ")
	}

	remaining := len(nonEmptyLabels) - maxLabels
	return fmt.Sprintf(
		"%s, +%d more",
		strings.Join(nonEmptyLabels[:maxLabels], ", "),
		remaining,
	)
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// GetDuration calculates and returns the duration the workflow, based on the given timestamps.
func GetDuration(
	runStartedAt, runFinishedAt, runStoppedAt *string,
) (any, error) {
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
	} else if runStoppedAt != nil {
		endTime, err = datautils.FromIsoToTimestamp(*runStoppedAt)
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
	switch status {
	case "finished", "failed":
		verb = "has"
	case "created", "stopped", "queued", "deleted":
		verb = "has been"
	case "running", "pending":
		verb = "is"
	default:
		return "", fmt.Errorf("unrecognised status %s", status)
	}

	return fmt.Sprintf("%s %s %s", workflow, verb, status), nil
}
