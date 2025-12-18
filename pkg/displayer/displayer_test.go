/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package displayer

import (
	"bytes"
	"fmt"
	"reanahub/reana-client-go/pkg/datautils"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

func TestDisplayTable(t *testing.T) {
	tests := map[string]struct {
		headers []string
		rows    [][]any
	}{
		"uppercase headers": {
			headers: []string{"HEADER1", "HEADER2"},
		},
		"lowercase headers": {
			headers: []string{"header1", "header2"},
		},
		"one row": {
			headers: []string{"HEADER1", "HEADER2"},
			rows: [][]any{
				{1, "test"},
			},
		},
		"many rows": {
			headers: []string{"HEADER1", "header2", "HEADER3"},
			rows: [][]any{
				{1, "test", true},
				{2, "test2", false},
				{3, "test3", true},
				{4.5, "this row is built different", false},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			DisplayTable(test.headers, test.rows, buf)
			result := buf.String()

			lines := datautils.SplitLinesNoEmpty(result)
			if len(lines) != len(test.rows)+1 {
				t.Fatalf(
					"Expected %d table lines, got %d",
					len(test.rows)+1,
					len(lines),
				)
			}

			for _, header := range test.headers {
				header = strings.ToUpper(header)
				if !strings.Contains(lines[0], header) {
					t.Fatalf(
						"Expected to contain header: \"%s\", got: \"%s\"",
						header,
						result,
					)
				}
			}

			for i, row := range test.rows {
				for j, col := range row {
					if !strings.Contains(lines[i+1], fmt.Sprintf("%v", col)) {
						t.Fatalf(
							"Expected to contain row %d column %d: '%s', got: '%s'",
							i,
							j,
							col,
							result,
						)
					}
				}
			}
		})
	}
}

func TestDisplayJsonOutput(t *testing.T) {
	tests := map[string]struct {
		arg      any
		expected string
	}{
		"string": {
			arg: "test", expected: "\"test\"\n",
		},
		"int": {
			arg: 1, expected: "1\n",
		},
		"float": {
			arg: 1.1, expected: "1.1\n",
		},
		"array of strings": {
			arg: []string{
				"test",
				"test2",
			}, expected: "[\n  \"test\",\n  \"test2\"\n]\n",
		},
		"array of any": {
			arg: []any{1, "test"}, expected: "[\n  1,\n  \"test\"\n]\n",
		},
		"map of strings": {
			arg: map[string]string{
				"key": "value",
			}, expected: "{\n  \"key\": \"value\"\n}\n",
		},
		"map of any": {
			arg: map[string]any{
				"key":  1,
				"key2": true,
			}, expected: "{\n  \"key\": 1,\n  \"key2\": true\n}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := DisplayJsonOutput(test.arg, buf)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			result := buf.String()
			if result != test.expected {
				t.Fatalf("Expected: '%s', got: '%s'", test.expected, result)
			}
		})
	}
}

func TestDisplayMessage(t *testing.T) {
	tests := map[string]struct {
		msg      string
		msgType  MessageType
		indented bool
		expected string
	}{
		"success": {
			msg:      "test success",
			msgType:  Success,
			indented: false,
			expected: text.Colors{
				text.FgGreen,
				text.Bold,
			}.Sprint(
				"==> SUCCESS: ",
			) + "test success\n",
		},
		"success indented": {
			msg:      "test success indented",
			msgType:  Success,
			indented: true,
			expected: text.Colors{
				text.FgGreen,
				text.Bold,
			}.Sprint(
				"  -> SUCCESS: ",
			) + "test success indented\n",
		},
		"error": {
			msg:      "test error",
			msgType:  Error,
			indented: false,
			expected: text.Colors{
				text.FgRed,
				text.Bold,
			}.Sprint(
				"==> ERROR: ",
			) + "test error\n",
		},
		"info": {
			msg:      "test info",
			msgType:  Info,
			indented: false,
			expected: text.Bold.Sprint(
				"==> test info\n",
			), // should be bold without any color
		},
		"info indented": {
			msg:      "test info indented",
			msgType:  Info,
			indented: true,
			expected: text.Colors{
				text.FgCyan,
				text.Bold,
			}.Sprint(
				"  -> INFO: ",
			) + "test info indented\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			DisplayMessage(test.msg, test.msgType, test.indented, buf)
			result := buf.String()
			if result != test.expected {
				t.Fatalf("Expected: \"%s\", got: \"%s\"", test.expected, result)
			}
		})
	}
}

func TestPrintColorable(t *testing.T) {
	tests := map[string]struct {
		str          string
		colorOptions []text.Color
		expected     string
	}{
		"simple text": {
			str: "test", expected: "test",
		},
		"bold": {
			str: "test\ntest", colorOptions: []text.Color{text.Bold},
			expected: text.Bold.Sprint("test\ntest"),
		},
		"bold and colored": {
			str: "test", colorOptions: []text.Color{text.Bold, text.FgRed},
			expected: text.Colors{text.Bold, text.FgRed}.Sprint("test"),
		},
		"multiple options": {
			str: "test", colorOptions: []text.Color{text.ReverseVideo, text.FgRed, text.Underline, text.Italic},
			expected: text.Colors{
				text.ReverseVideo,
				text.FgRed,
				text.Underline,
				text.Italic,
			}.Sprint(
				"test",
			),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			PrintColorable(test.str, buf, test.colorOptions...)
			result := buf.String()
			if result != test.expected {
				t.Fatalf("Expected: '%s', got: '%s'", test.expected, result)
			}
		})
	}
}
