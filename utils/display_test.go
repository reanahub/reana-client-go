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
	tests := []struct {
		msg      string
		msgType  MessageType
		indented bool
		expected string
	}{
		{
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
		{
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
		{
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
		{
			msg:      "test info",
			msgType:  Info,
			indented: false,
			expected: text.Bold.Sprint("==> test info\n"), // should be bold without any color
		},
		{
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

	for _, test := range tests {
		buf := new(bytes.Buffer)
		DisplayMessage(test.msg, test.msgType, test.indented, buf)
		result := buf.String()
		if result != test.expected {
			t.Fatalf("Expected: \"%s\", got: \"%s\"", test.expected, result)
		}
	}
}
