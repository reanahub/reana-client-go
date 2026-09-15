/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package tester

import (
	"crypto/md5" // MD5 is retained for Python-client feature parity.
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	messages "github.com/cucumber/messages/go/v34"
)

const workflowTimeFormat = "2006-01-02T15:04:05"

var dimensionPattern = regexp.MustCompile(
	`^(\d+(?:\.\d+)?)\s*([A-Za-z]*)$`,
)

func stripQuotes(value string) string {
	return strings.Trim(value, `"`)
}

func cleanWorkspacePath(value string) string {
	return strings.TrimPrefix(
		strings.TrimPrefix(stripQuotes(value), "./"),
		"/",
	)
}

func workspaceDownloadPath(value string) string {
	// The API route already supplies the slash before the file name.
	return strings.TrimLeft(stripQuotes(value), "/")
}

func humanReadableToBytes(value string) (int64, error) {
	matches := dimensionPattern.FindStringSubmatch(value)
	if matches == nil {
		return 0, fmt.Errorf("unable to parse %q", value)
	}
	size, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse %q: %w", value, err)
	}
	units := map[string]float64{
		"":      1,
		"bytes": 1,
		"B":     1,
		"KiB":   1 << 10,
		"MiB":   1 << 20,
		"GiB":   1 << 30,
		"TiB":   1 << 40,
		"PiB":   1 << 50,
	}
	multiplier, ok := units[matches[2]]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q", matches[2])
	}
	return int64(size * multiplier), nil
}

func fileInWorkspace(
	workflow,
	fileName string,
	dataFetcher DataFetcher,
) (bool, error) {
	files, err := dataFetcher.DiskUsage(workflow, false)
	if err != nil {
		return false, err
	}
	for _, file := range files {
		if file.Name == fileName || file.Name == "/"+fileName {
			return true, nil
		}
	}
	return false, nil
}

func singleFile(
	workflow,
	fileName string,
	dataFetcher DataFetcher,
) (FileInfo, error) {
	files, err := dataFetcher.Files(workflow, fileName)
	if err != nil {
		return FileInfo{}, err
	}
	if len(files) != 1 {
		return FileInfo{}, fmt.Errorf(
			"the specified file name (%s) is not in the workspace",
			fileName,
		)
	}
	return files[0], nil
}

func totalWorkspaceSize(
	workflow string,
	dataFetcher DataFetcher,
) (int64, error) {
	files, err := dataFetcher.DiskUsage(workflow, true)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, fmt.Errorf("workspace disk usage response is empty")
	}
	return files[0].Size, nil
}

func engineSpecificText(content json.RawMessage) string {
	if len(content) == 0 || string(content) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(content, &text); err == nil {
		return text
	}
	// Engines such as Yadage return structured DAG metadata instead of text.
	// Search its serialised JSON so engine-log assertions can still match it,
	// accepting that matches may come from structure rather than log output.
	return string(content)
}

func logsContain(logs WorkflowLogs, content string) bool {
	if strings.Contains(
		logs.WorkflowLogs+engineSpecificText(logs.EngineSpecific),
		content,
	) {
		return true
	}
	for _, job := range logs.JobLogs {
		if strings.Contains(job.Logs, content) {
			return true
		}
	}
	return false
}

func findJob(logs WorkflowLogs, stepName string) (JobLog, error) {
	var selected JobLog
	selectedKey := ""
	found := false
	for key, job := range logs.JobLogs {
		if job.JobName != stepName {
			continue
		}
		// Select the most recently started retry or scatter job. This approximates
		// the Python client selecting the last entry from the server's
		// creation-ordered job map. Timestamps use a fixed-width format, so they
		// compare chronologically as strings. Job IDs are random UUIDs and serve
		// only as deterministic tie-breakers.
		if !found || job.StartedAt > selected.StartedAt ||
			(job.StartedAt == selected.StartedAt && key > selectedKey) {
			selected = job
			selectedKey = key
			found = true
		}
	}
	if found {
		return selected, nil
	}
	return JobLog{}, fmt.Errorf("the specified step name is invalid")
}

func parseWorkflowTime(value string) (time.Time, error) {
	parsed, err := time.Parse(workflowTimeFormat, value)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid workflow timestamp %q: %w",
			value,
			err,
		)
	}
	return parsed, nil
}

func checkWorkflowCompleted(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, _ map[string]string) error {
		status, err := dataFetcher.Status(workflow)
		if err != nil {
			return err
		}
		if status.Status != "finished" && status.Status != "failed" {
			return &stepSkipped{reason: fmt.Sprintf(
				"the execution of workflow %q has not completed; its status is %q",
				workflow,
				status.Status,
			)}
		}
		return nil
	}
}

func checkWorkflowStatus(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		expected := stripQuotes(arguments["status_workflow"])
		status, err := dataFetcher.Status(workflow)
		if err != nil {
			return err
		}
		if status.Status != expected {
			return &stepSkipped{reason: fmt.Sprintf(
				"workflow %q is not in %q status; its status is %q",
				workflow,
				expected,
				status.Status,
			)}
		}
		return nil
	}
}

func assertWorkflowStatus(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		expected := stripQuotes(arguments["status_workflow"])
		status, err := dataFetcher.Status(workflow)
		if err != nil {
			return err
		}
		if status.Status != expected {
			return fmt.Errorf(
				"workflow %q is not %q; its status is %q",
				workflow,
				expected,
				status.Status,
			)
		}
		return nil
	}
}

func assertAllOutputs(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, _ map[string]string) error {
		specification, err := dataFetcher.Specification(workflow)
		if err != nil {
			return err
		}
		outputs := append(
			append([]string{}, specification.OutputFiles...),
			specification.OutputDirectories...,
		)
		// Include both output kinds as documented. The Python implementation
		// currently drops directories when files are present due to expression
		// precedence; preserving that bug would make the vocabulary misleading.
		for _, output := range outputs {
			included, err := fileInWorkspace(workflow, output, dataFetcher)
			if err != nil {
				return err
			}
			if !included {
				return fmt.Errorf("workspace does not contain %q", output)
			}
		}
		return nil
	}
}

func assertWorkspaceContains(
	dataFetcher DataFetcher,
	expected bool,
) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		// Normalise quotes for every filename alias. The Python implementation
		// misses this for its unquoted-pattern alias, contrary to its own docs.
		fileName := stripQuotes(arguments["filename"])
		included, err := fileInWorkspace(workflow, fileName, dataFetcher)
		if err != nil {
			return err
		}
		if included != expected {
			if expected {
				return fmt.Errorf("workspace does not contain %q", fileName)
			}
			return fmt.Errorf("workspace contains %q", fileName)
		}
		return nil
	}
}

func assertLogsContain(dataFetcher DataFetcher, source string) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		content := arguments["content"]
		logs, err := dataFetcher.Logs(workflow, nil)
		if err != nil {
			return err
		}
		contains := false
		switch source {
		case "all":
			contains = logsContain(logs, content)
		case "engine":
			contains = strings.Contains(
				logs.WorkflowLogs+engineSpecificText(logs.EngineSpecific),
				content,
			)
		case "job":
			for _, job := range logs.JobLogs {
				if strings.Contains(job.Logs, content) {
					contains = true
					break
				}
			}
		}
		if !contains {
			return fmt.Errorf("%s logs do not contain %q", source, content)
		}
		return nil
	}
}

func assertStepLogsContain(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		stepName := stripQuotes(arguments["step_name"])
		content := arguments["content"]
		logs, err := dataFetcher.Logs(workflow, []string{stepName})
		if err != nil {
			return err
		}
		job, err := findJob(logs, stepName)
		if err != nil {
			return err
		}
		if !strings.Contains(job.Logs, content) {
			return fmt.Errorf(
				"logs for step %q do not contain %q",
				stepName,
				content,
			)
		}
		return nil
	}
}

func assertFileContains(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		fileName := workspaceDownloadPath(arguments["filename"])
		file, err := dataFetcher.Download(workflow, fileName)
		if err != nil {
			return err
		}
		if file.IsArchive {
			return &stepSkipped{
				reason: "file content tests are not supported for archive files",
			}
		}
		if !utf8.Valid(file.Content) {
			return fmt.Errorf("file %q does not contain valid UTF-8", fileName)
		}
		content := arguments["content"]
		if !strings.Contains(string(file.Content), content) {
			return fmt.Errorf("file %q does not contain %q", fileName, content)
		}
		return nil
	}
}

func assertFileChecksum(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		algorithm := strings.ToLower(stripQuotes(arguments["algorithm"]))
		fileName := workspaceDownloadPath(arguments["filename"])
		expected := strings.ToLower(stripQuotes(arguments["checksum"]))
		file, err := dataFetcher.Download(workflow, fileName)
		if err != nil {
			return err
		}
		var checksum string
		switch algorithm {
		case "sha256":
			sum := sha256.Sum256(file.Content)
			checksum = hex.EncodeToString(sum[:])
		case "sha512":
			sum := sha512.Sum512(file.Content)
			checksum = hex.EncodeToString(sum[:])
		case "md5":
			sum := md5.Sum(file.Content)
			checksum = hex.EncodeToString(sum[:])
		case "adler32":
			checksum = strconv.FormatUint(
				uint64(adler32.Checksum(file.Content)),
				16,
			)
			expected = strings.TrimPrefix(expected, "0x")
		default:
			return fmt.Errorf(
				"unsupported checksum algorithm %q; supported algorithms: sha256, sha512, md5, adler32",
				algorithm,
			)
		}
		if checksum != expected {
			return fmt.Errorf(
				"checksum of file %q is not %q; actual checksum: %q",
				fileName,
				expected,
				checksum,
			)
		}
		return nil
	}
}

func assertWorkflowDuration(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		minutes, err := strconv.ParseFloat(
			stripQuotes(arguments["n_minutes"]),
			64,
		)
		if err != nil {
			return err
		}
		status, err := dataFetcher.Status(workflow)
		if err != nil {
			return err
		}
		startedAt, err := parseWorkflowTime(status.RunStartedAt)
		if err != nil {
			return err
		}
		finishedAt, err := parseWorkflowTime(status.RunFinishedAt)
		if err != nil {
			return err
		}
		duration := finishedAt.Sub(startedAt).Minutes()
		if duration >= minutes {
			return fmt.Errorf(
				"workflow took %.2f minutes, expected less than %.2f minutes",
				duration,
				minutes,
			)
		}
		return nil
	}
}

func assertStepDuration(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		stepName := stripQuotes(arguments["step_name"])
		minutes, err := strconv.ParseFloat(
			stripQuotes(arguments["n_minutes"]),
			64,
		)
		if err != nil {
			return err
		}
		logs, err := dataFetcher.Logs(workflow, []string{stepName})
		if err != nil {
			return err
		}
		job, err := findJob(logs, stepName)
		if err != nil {
			return err
		}
		startedAt, err := parseWorkflowTime(job.StartedAt)
		if err != nil {
			return err
		}
		finishedAt, err := parseWorkflowTime(job.FinishedAt)
		if err != nil {
			return err
		}
		duration := finishedAt.Sub(startedAt).Minutes()
		if duration >= minutes {
			return fmt.Errorf(
				"step %q took %.2f minutes, expected less than %.2f minutes",
				stepName,
				duration,
				minutes,
			)
		}
		return nil
	}
}

func assertExactFileSize(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		expected, err := humanReadableToBytes(stripQuotes(arguments["dim"]))
		if err != nil {
			return err
		}
		fileName := cleanWorkspacePath(arguments["filename"])
		file, err := singleFile(workflow, fileName, dataFetcher)
		if err != nil {
			return err
		}
		if file.Size != expected {
			return fmt.Errorf(
				"size of file %q is %d bytes, expected %d bytes",
				fileName,
				file.Size,
				expected,
			)
		}
		return nil
	}
}

func assertApproximateFileSize(dataFetcher DataFetcher) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		lower, err := humanReadableToBytes(stripQuotes(arguments["dim1"]))
		if err != nil {
			return err
		}
		upper, err := humanReadableToBytes(stripQuotes(arguments["dim2"]))
		if err != nil {
			return err
		}
		if lower > upper {
			lower, upper = upper, lower
		}
		fileName := cleanWorkspacePath(arguments["filename"])
		file, err := singleFile(workflow, fileName, dataFetcher)
		if err != nil {
			return err
		}
		if file.Size < lower || file.Size > upper {
			return fmt.Errorf(
				"size of file %q is %d bytes, expected between %d and %d bytes",
				fileName,
				file.Size,
				lower,
				upper,
			)
		}
		return nil
	}
}

func assertWorkspaceSize(
	dataFetcher DataFetcher,
	maximum bool,
) stepHandler {
	return func(workflow string, arguments map[string]string) error {
		expected, err := humanReadableToBytes(stripQuotes(arguments["dim"]))
		if err != nil {
			return err
		}
		actual, err := totalWorkspaceSize(workflow, dataFetcher)
		if err != nil {
			return err
		}
		if maximum && actual > expected {
			return fmt.Errorf(
				"workspace size is %d bytes, expected no more than %d bytes",
				actual,
				expected,
			)
		}
		if !maximum && actual < expected {
			return fmt.Errorf(
				"workspace size is %d bytes, expected at least %d bytes",
				actual,
				expected,
			)
		}
		return nil
	}
}

func stepDefinitions(dataFetcher DataFetcher) []stepDefinition {
	action := messages.PickleStepType_ACTION
	outcome := messages.PickleStepType_OUTCOME
	return []stepDefinition{
		newStepDefinition(
			action,
			"the workflow execution completes",
			checkWorkflowCompleted(dataFetcher),
		),
		newStepDefinition(
			action,
			"the workflow is {status_workflow}",
			checkWorkflowStatus(dataFetcher),
		),
		newStepDefinition(
			action,
			"the workflow status is {status_workflow}",
			checkWorkflowStatus(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the workflow should be {status_workflow}",
			assertWorkflowStatus(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the workflow status should be {status_workflow}",
			assertWorkflowStatus(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the outputs should be included in the workspace",
			assertAllOutputs(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"all the outputs should be included in the workspace",
			assertAllOutputs(dataFetcher),
		),
		newStepDefinition(
			outcome,
			`the workspace should include "{filename}"`,
			assertWorkspaceContains(dataFetcher, true),
		),
		newStepDefinition(
			outcome,
			`the workspace should contain "{filename}"`,
			assertWorkspaceContains(dataFetcher, true),
		),
		newStepDefinition(
			outcome,
			"{filename} should be in the workspace",
			assertWorkspaceContains(dataFetcher, true),
		),
		newStepDefinition(
			outcome,
			"the workspace should not include {filename}",
			assertWorkspaceContains(dataFetcher, false),
		),
		newStepDefinition(
			outcome,
			"the workspace should not contain {filename}",
			assertWorkspaceContains(dataFetcher, false),
		),
		newStepDefinition(
			outcome,
			"{filename} should not be in the workspace",
			assertWorkspaceContains(dataFetcher, false),
		),
		newStepDefinition(
			outcome,
			`the logs should contain "{content}"`,
			assertLogsContain(dataFetcher, "all"),
		),
		newStepDefinition(
			outcome,
			`the engine logs should contain "{content}"`,
			assertLogsContain(dataFetcher, "engine"),
		),
		newStepDefinition(
			outcome,
			`the job logs should contain "{content}"`,
			assertLogsContain(dataFetcher, "job"),
		),
		newStepDefinition(
			outcome,
			`the job logs for the step {step_name} should contain "{content}"`,
			assertStepLogsContain(dataFetcher),
		),
		newStepDefinition(
			outcome,
			`the job logs for the {step_name} step should contain "{content}"`,
			assertStepLogsContain(dataFetcher),
		),
		newStepDefinition(
			outcome,
			`the file {filename} should include "{content}"`,
			assertFileContains(dataFetcher),
		),
		newStepDefinition(
			outcome,
			`the file {filename} should contain "{content}"`,
			assertFileContains(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the {algorithm} checksum of the file {filename} should be {checksum}",
			assertFileChecksum(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the workflow run duration should be less than {n_minutes} minutes",
			assertWorkflowDuration(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the duration of the step {step_name} should be less than {n_minutes} minutes",
			assertStepDuration(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the size of the file {filename} should be exactly {dim}",
			assertExactFileSize(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the size of the file {filename} should be between {dim1} and {dim2}",
			assertApproximateFileSize(dataFetcher),
		),
		newStepDefinition(
			outcome,
			"the workspace size should be less than {dim}",
			assertWorkspaceSize(dataFetcher, true),
		),
		newStepDefinition(
			outcome,
			"the workspace size should be more than {dim}",
			assertWorkspaceSize(dataFetcher, false),
		),
	}
}
