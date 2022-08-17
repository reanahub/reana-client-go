/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package displayer gives utility functions to display messages, tables, amongst others.
package displayer

import (
	"encoding/json"
	"fmt"
	"io"
	"reanahub/reana-client-go/pkg/config"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// MessageType represents a type of message to be displayed (e.g. success and error).
type MessageType int

const (
	Success MessageType = iota
	Warning
	Error
	Info
)

// String return the string representation of the MessageType.
func (m MessageType) String() string {
	return []string{"SUCCESS", "WARNING", "ERROR", "INFO"}[m]
}

// Color returns a text color according to the MessageType.
func (m MessageType) Color() text.Color {
	return []text.Color{text.FgGreen, text.FgYellow, text.FgRed, text.FgCyan}[m]
}

// JobStatusToColor Maps the different job status to a matching color. Can be used with PrintColorable.
var JobStatusToColor = map[string]text.Color{
	"failed":   text.FgRed,
	"finished": text.FgGreen,
	"running":  text.FgCyan,
}

// DisplayTable takes a header and the respective rows, and formats them in a table.
// Instead of writing to stdout, it uses the provided io.Writer.
func DisplayTable[T any](header []string, rows [][]T, out io.Writer) {
	// Convert to table.Row type
	rowList := make([]table.Row, len(rows))
	for i, row := range rows {
		tableRow := make(table.Row, len(row))
		for j, cell := range row {
			tableRow[j] = cell
		}
		rowList[i] = tableRow
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

// DisplayJsonOutput displays the given output in a JSON format. The output should be compatible with json.Marshal.
// Instead of writing to stdout, it uses the provided io.Writer.
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

// DisplayMessage takes a message, a messageType (e.g. success or error) and displays it according to the color
// associated with the messageType and whether it is indented or not.
func DisplayMessage(message string, messageType MessageType, indented bool, out io.Writer) {
	prefix := config.LeadingMark
	if indented {
		prefix = "  ->"
	}

	if messageType == Info && !indented {
		msg := text.Bold.Sprintf("%s %s\n", prefix, message)
		fmt.Fprint(out, msg)
		return
	}

	PrintColorable(
		fmt.Sprintf("%s %s: ", prefix, messageType),
		out,
		messageType.Color(),
		text.Bold,
	)
	fmt.Fprintln(out, message)
}

// PrintColorable prints a colorable string, according to the colorOptions provided.
func PrintColorable(str string, out io.Writer, colorOptions ...text.Color) {
	var colors text.Colors
	colors = append(colors, colorOptions...)
	fmt.Fprint(out, colors.Sprint(str))
}
