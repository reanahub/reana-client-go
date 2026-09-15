/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package tester

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type downloadMockFetcher struct {
	*mockFetcher
	download DownloadedFile
	err      error
}

func (f *downloadMockFetcher) Download(
	_, _ string,
) (DownloadedFile, error) {
	return f.download, f.err
}

func TestWorkspaceDownloadPathUsesRelativePath(t *testing.T) {
	if got := workspaceDownloadPath(`"/result.txt"`); got != "result.txt" {
		t.Errorf("got %q, want result.txt", got)
	}
}

func TestHumanReadableToBytesRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{
		"invalid",
		// Overflow float64 to exercise the ParseFloat error branch.
		strings.Repeat("9", 400),
		"1 KB",
	} {
		if _, err := humanReadableToBytes(value); err == nil {
			t.Errorf("expected %q to be rejected", value)
		}
	}
}

func TestLogsContainStructuredEngineDataAndJobOutput(t *testing.T) {
	logs := WorkflowLogs{
		EngineSpecific: json.RawMessage(`{"dag":{"nodes":[]}}`),
		JobLogs: map[string]JobLog{
			"1": {Logs: "job output"},
		},
	}
	if !strings.Contains(engineSpecificText(logs.EngineSpecific), `"dag"`) {
		t.Error("engine-specific JSON was not rendered")
	}
	if engineSpecificText(nil) != "" {
		t.Error("empty engine-specific data was not ignored")
	}
	if !logsContain(logs, "job output") {
		t.Error("job log content was not found")
	}
	if logsContain(logs, "missing") {
		t.Error("unexpected log content was found")
	}
}

func TestParseWorkflowTimeRejectsInvalidValue(t *testing.T) {
	if _, err := parseWorkflowTime("not-a-timestamp"); err == nil {
		t.Error("expected invalid timestamp error")
	}
}

func TestAssertWorkflowStatusReportsMismatch(t *testing.T) {
	fetcher := newMockFetcher()
	handler := assertWorkflowStatus(fetcher)
	if err := handler(
		"analysis.1",
		map[string]string{"status_workflow": `"failed"`},
	); err == nil {
		t.Error("expected workflow status mismatch")
	}
	if err := handler(
		"analysis.1",
		map[string]string{"status_workflow": `"finished"`},
	); err != nil {
		t.Fatalf("unexpected workflow status error: %v", err)
	}
}

func TestAssertFileContainsRejectsUnsupportedContent(t *testing.T) {
	t.Run("archive", func(t *testing.T) {
		fetcher := &downloadMockFetcher{
			mockFetcher: newMockFetcher(),
			download:    DownloadedFile{IsArchive: true},
		}
		err := assertFileContains(fetcher)(
			"analysis.1",
			map[string]string{
				"filename": `"results/archive.zip"`,
				"content":  "value",
			},
		)
		var skipped *stepSkipped
		if !errors.As(err, &skipped) {
			t.Fatalf("expected archive assertion to be skipped, got %v", err)
		}
	})

	t.Run("non-UTF-8", func(t *testing.T) {
		fetcher := &downloadMockFetcher{
			mockFetcher: newMockFetcher(),
			download:    DownloadedFile{Content: []byte{0xff}},
		}
		err := assertFileContains(fetcher)(
			"analysis.1",
			map[string]string{
				"filename": `"results/data.bin"`,
				"content":  "value",
			},
		)
		if err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
			t.Fatalf("expected UTF-8 error, got %v", err)
		}
	})

}

func TestAssertFileContainsReportsContentMismatch(t *testing.T) {
	fetcher := &downloadMockFetcher{
		mockFetcher: newMockFetcher(),
		download:    DownloadedFile{Content: []byte("present")},
	}
	if err := assertFileContains(fetcher)(
		"analysis.1",
		map[string]string{
			"filename": `"results/message.txt"`,
			"content":  "missing",
		},
	); err == nil {
		t.Error("expected file content mismatch")
	}
}

func TestAssertFileChecksumReportsMismatch(t *testing.T) {
	if err := assertFileChecksum(newMockFetcher())(
		"analysis.1",
		map[string]string{
			"algorithm": "sha256",
			"filename":  `"results/message.txt"`,
			"checksum":  "wrong",
		},
	); err == nil {
		t.Error("expected checksum mismatch")
	}
}
