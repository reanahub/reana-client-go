/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"golang.org/x/exp/slices"
)

type FormatFilter struct {
	column     string
	value      string
	filterRows bool
}

// ParseFilterParameters parses a list of filters in the format 'filter=value'.
// The 'status' filters are returned as a slice of strings, while the remaining filters are returned as a JSON string.
// Every given filter must be included in filterNames.
func ParseFilterParameters(filter []string, filterNames []string) ([]string, string, error) {
	searchFilters := make(map[string][]string)
	var statusFilters []string

	for _, value := range filter {
		filterName, filterValue, err := GetFilterNameAndValue(value)
		if err != nil {
			return nil, "", err
		}

		if !slices.Contains(filterNames, filterName) {
			return nil, "", fmt.Errorf("filter %s is not valid", filterName)
		}

		if filterName == "status" && !slices.Contains(GetRunStatuses(true), filterValue) {
			return nil, "", fmt.Errorf("input status value %s is not valid. ", filterValue)
		}

		if filterName == "status" {
			statusFilters = append(statusFilters, filterValue)
		} else {
			searchFilters[filterName] = append(searchFilters[filterName], filterValue)
		}
	}

	searchFiltersString := ""
	if len(searchFilters) > 0 {
		searchFiltersByteArray, err := json.Marshal(searchFilters)
		if err != nil {
			return nil, "", err
		}
		searchFiltersString = string(searchFiltersByteArray)
	}

	return statusFilters, searchFiltersString, nil
}

// GetFilterNameAndValue parses a filter in the format 'filter=value' and returns them.
func GetFilterNameAndValue(filter string) (string, string, error) {
	if !strings.Contains(filter, "=") {
		return "", "", errors.New(
			"wrong input format. Please use --filter filter_name=filter_value",
		)
	}

	filterNameAndValue := strings.SplitN(filter, "=", 2)
	filterName := strings.ToLower(filterNameAndValue[0])
	filterValue := filterNameAndValue[1]
	return filterName, filterValue, nil
}

// ParseFormatParameters parses a list of formatOptions to a slice of FormatFilter.
// If the format option has a filter, that will be the value in the struct and the filterRows boolean will be true.
func ParseFormatParameters(formatOptions []string) []FormatFilter {
	var parsedFilters []FormatFilter
	for _, filter := range formatOptions {
		filterNameAndValue := strings.SplitN(filter, "=", 2)
		formatFilter := FormatFilter{column: filterNameAndValue[0], filterRows: false}
		if len(filterNameAndValue) >= 2 {
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
