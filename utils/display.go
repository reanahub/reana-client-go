/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type MessageType int

const (
	Success MessageType = iota
	Warning
	Error
	Info
)

func (m MessageType) String() string {
	return []string{"SUCCESS", "WARNING", "ERROR", "INFO"}[m]
}

func (m MessageType) Color() text.Color {
	return []text.Color{text.FgGreen, text.FgYellow, text.FgRed, text.FgCyan}[m]
}

func DisplayTable(header []string, rows [][]any, out io.Writer) {
	// Convert to table.Row type
	rowList := make([]table.Row, len(rows))
	for i, r := range rows {
		rowList[i] = r
	}

	headerRow := make(table.Row, len(header))
	for i, h := range header {
		headerRow[i] = h
	}

	t := table.NewWriter()
	t.SetOutputMirror(out)
	t.AppendHeader(headerRow)
	t.AppendRows(rowList)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateHeader = false
	t.Render()
}

func DisplayJsonOutput(output any, out io.Writer) error {
	byteArray, err := json.MarshalIndent(output, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to display json output:\n%v", err)
	}

	_, err = fmt.Fprintln(out, string(byteArray))
	if err != nil {
		return err
	}
	return nil
}

func DisplayMessage(message string, messageType MessageType, indented bool, out io.Writer) {
	prefix := "==>"
	if indented {
		prefix = "  ->"
	}

	if messageType == Info && !indented {
		msg := text.Bold.Sprintf("%s %s\n", prefix, message)
		fmt.Fprint(out, msg)
		return
	}

	prefixTpl := colorizeMessage(
		fmt.Sprintf("%s %s: ", prefix, messageType),
		messageType,
	)

	fmt.Fprintf(out, "%s%s\n", prefixTpl, message)
}

func colorizeMessage(msg string, msgType MessageType) string {
	return text.Colors{msgType.Color(), text.Bold}.Sprint(msg)
}
