/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"bytes"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

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
			expected: text.Bold.Sprint("==> test info\n"), // should be bold without any color
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
