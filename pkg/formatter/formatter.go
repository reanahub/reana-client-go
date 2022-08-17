/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package formatter gives data structures and functions to handle formatting of tabular data.
package formatter

import (
	"fmt"
	"strings"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"golang.org/x/exp/slices"
)

// FormatFilter provides a centralized way of handling format options across the different commands.
type FormatFilter struct {
	column     string
	value      string
	filterRows bool // set to true if a value was provided and the rows should be filtered by this column.
}

// ParseFormatParameters parses a list of formatOptions to a slice of FormatFilter.
// If the format option has a filter, that will be the value in the struct and the filterRows boolean will be true.
func ParseFormatParameters(formatOptions []string, filterRows bool) []FormatFilter {
	var parsedFilters []FormatFilter
	for _, filter := range formatOptions {
		filterNameAndValue := strings.SplitN(filter, "=", 2)
		formatFilter := FormatFilter{column: filterNameAndValue[0], filterRows: false}
		if filterRows && len(filterNameAndValue) >= 2 {
			formatFilter.value = filterNameAndValue[1]
			formatFilter.filterRows = true
		}
		parsedFilters = append(parsedFilters, formatFilter)
	}
	return parsedFilters
}

// FormatDataFrame formats a dataFrame according to the formatFilters provided.
// The formatFilters can be previously obtained with ParseFormatParameters.
func FormatDataFrame(df dataframe.DataFrame, formatFilters []FormatFilter) dataframe.DataFrame {
	if len(formatFilters) == 0 {
		return df
	}

	var newCols []series.Series
	for _, filter := range formatFilters {
		newCols = append(newCols, df.Col(filter.column))
	}
	df = dataframe.New(newCols...)

	for _, filter := range formatFilters {
		if filter.filterRows {
			df = df.Filter(dataframe.F{
				Colname: filter.column, Comparator: series.Eq, Comparando: filter.value,
			})
		}
	}
	return df
}

// SortDataFrame sorts the given dataFrame according to the sortColumn and whether the order is reversed.
// The sortColumn must be included in the df header.
func SortDataFrame(
	df dataframe.DataFrame,
	sortColumn string,
	reverse bool,
) (dataframe.DataFrame, error) {
	sortColumn = strings.ToLower(sortColumn)
	if !slices.Contains(df.Names(), sortColumn) {
		return df, fmt.Errorf("column '%s' does not exist", sortColumn)
	}

	return df.Arrange(dataframe.Order{Colname: sortColumn, Reverse: reverse}), nil
}

// DataFrameToStringData converts a given dataFrame to a 2D slice of strings.
// Converts null values to "-".
func DataFrameToStringData(df dataframe.DataFrame) [][]string {
	data := df.Records()[1:] // Ignore col names
	for i, row := range data {
		for j := range row {
			if df.Elem(i, j).IsNA() {
				data[i][j] = "-"
			}
		}
	}
	return data
}

// FormatSessionURI takes the serverURL, its token and a path, and formats them into a session URI.
func FormatSessionURI(serverURL string, path string, token string) string {
	return serverURL + path + "?token=" + token
}
