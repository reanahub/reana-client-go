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

// Filters provides a centralized way of handling filters across the different commands.
type Filters struct {
	SingleFilterKeys   []string // names (keys) of the single value filters to be considered
	MultiFilterKeys    []string // names (keys) of the multi value filters to be considered
	singleValueFilters map[string]string
	multiValueFilters  map[string][]string
}

// NewFilters returns a new instance of Filters, with the given keys.
// singleFilterKeys are the filters with only one value at a time, while multiFilterKeys can accumulate values.
func NewFilters(singleFilterKeys []string, multiFilterKeys []string) Filters {
	return Filters{
		SingleFilterKeys:   singleFilterKeys,
		MultiFilterKeys:    multiFilterKeys,
		singleValueFilters: make(map[string]string),
		multiValueFilters:  make(map[string][]string),
	}
}

// AddFilters adds multiple filters, in the format 'key=value'. Works for both single and multi value filters.
func (f *Filters) AddFilters(filters []string) error {
	for _, filter := range filters {
		err := f.AddFilter(filter)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddFilter adds a filter, in the format 'key=value'. Works for both single and multi value filters.
func (f *Filters) AddFilter(filter string) error {
	key, value, err := f.getKeyAndValue(filter)
	if err != nil {
		return err
	}

	if slices.Contains(f.SingleFilterKeys, key) {
		f.singleValueFilters[key] = value
	} else if slices.Contains(f.MultiFilterKeys, key) {
		f.multiValueFilters[key] = append(f.multiValueFilters[key], value)
	} else {
		return fmt.Errorf(
			"filter key '%s' is not valid\nAvailable filters are '%s'",
			key,
			strings.Join(append(f.SingleFilterKeys, f.MultiFilterKeys...), "', '"),
		)
	}
	return nil
}

// GetSingle returns the value of a single value filter.
func (f Filters) GetSingle(key string) (string, error) {
	if !slices.Contains(f.SingleFilterKeys, key) {
		return "", fmt.Errorf(
			"'%s' is not a valid single value filter\nAvailable filters are '%s'",
			key,
			strings.Join(f.SingleFilterKeys, "', '"),
		)
	}

	return f.singleValueFilters[key], nil
}

// GetMulti returns a slice with the values of a multi value filter.
func (f Filters) GetMulti(key string) ([]string, error) {
	if !slices.Contains(f.MultiFilterKeys, key) {
		return []string{}, fmt.Errorf(
			"'%s' is not a valid multi value filter\nAvailable filters are '%s'",
			key,
			strings.Join(f.MultiFilterKeys, "', '"),
		)
	}

	return f.multiValueFilters[key], nil
}

// GetJson gets a JSON string with the filters specified in keys.
func (f Filters) GetJson(keys []string) (string, error) {
	jsonMap := make(map[string]any)
	for _, key := range keys {
		var (
			value  any
			exists bool
		)
		if slices.Contains(f.SingleFilterKeys, key) {
			value, exists = f.singleValueFilters[key]
		} else if slices.Contains(f.MultiFilterKeys, key) {
			value, exists = f.multiValueFilters[key]
		} else {
			return "", fmt.Errorf(
				"filter key '%s' is not valid\nAvailable filters are '%s'",
				key,
				strings.Join(append(f.SingleFilterKeys, f.MultiFilterKeys...), "', '"),
			)
		}
		if exists {
			jsonMap[key] = value
		}
	}

	jsonString := ""
	if len(jsonMap) > 0 {
		searchFiltersByteArray, err := json.Marshal(jsonMap)
		if err != nil {
			return "", err
		}
		jsonString = string(searchFiltersByteArray)
	}

	return jsonString, nil
}

// ValidateValues validates a given filter key, by comparing its value(s) with the possibleValues provided.
func (f Filters) ValidateValues(key string, possibleValues []string) error {
	if slices.Contains(f.SingleFilterKeys, key) {
		value, exists := f.singleValueFilters[key]
		if exists && !slices.Contains(possibleValues, value) {
			return fmt.Errorf(
				"'%s' is not a valid value for the filter '%s'\nAvailable values are '%s'",
				value, key,
				strings.Join(possibleValues, "', '"),
			)
		}
	} else if slices.Contains(f.MultiFilterKeys, key) {
		values, exists := f.multiValueFilters[key]
		if !exists {
			return nil
		}
		for _, value := range values {
			if !slices.Contains(possibleValues, value) {
				return fmt.Errorf(
					"'%s' is not a valid value for the filter '%s'\nAvailable values are '%s'",
					value, key,
					strings.Join(possibleValues, "', '"),
				)
			}
		}
	} else {
		return fmt.Errorf(
			"filter key '%s' is not valid\nAvailable filters are '%s'",
			key,
			strings.Join(append(f.SingleFilterKeys, f.MultiFilterKeys...), "', '"),
		)
	}
	return nil
}

// getKeyAndValue parses a filter in the format 'filter=value' and returns them.
func (f Filters) getKeyAndValue(filter string) (string, string, error) {
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

type FormatFilter struct {
	column     string
	value      string
	filterRows bool
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
