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
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

func DisplayTable(header []string, rows [][]any) {
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
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(headerRow)
	t.AppendRows(rowList)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateHeader = false
	t.Render()
}

func DisplayJsonOutput(output any) {
	byteArray, err := json.MarshalIndent(output, "", "  ")

	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	fmt.Println(string(byteArray))
}
