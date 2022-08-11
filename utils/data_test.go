/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

func TestFormatDataFrame(t *testing.T) {
	tests := map[string]struct {
		df            dataframe.DataFrame
		formatFilters []FormatFilter
	}{
		"no format": {
			df: dataframe.New(
				series.New([]string{"a", "b"}, series.String, "col1"),
				series.New([]int{1, 2}, series.Int, "col2"),
			),
		},
		"format without filters": {
			df: dataframe.New(
				series.New([]string{"a", "b"}, series.String, "col1"),
				series.New([]int{1, 2}, series.Int, "col2"),
			),
			formatFilters: []FormatFilter{{column: "col2"}},
		},
		"format with filters": {
			df: dataframe.New(
				series.New([]string{"a", "b"}, series.String, "col1"),
				series.New([]int{1, 2}, series.Int, "col2"),
			),
			formatFilters: []FormatFilter{{column: "col2", filterRows: true, value: "2"}},
		},
		"multiple format filters": {
			df: dataframe.New(
				series.New([]string{"a", "b", "c"}, series.String, "col1"),
				series.New([]int{1, 2, 2}, series.Int, "col2"),
				series.New([]bool{true, false, true}, series.Bool, "col3"),
				series.New([]float64{1.0, 2.0, 3.5}, series.Float, "col4"),
			),
			formatFilters: []FormatFilter{
				{column: "col1"},
				{column: "col2", filterRows: true, value: "2"},
				{column: "col3", filterRows: true, value: "false"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			df := FormatDataFrame(test.df, test.formatFilters)
			if len(test.formatFilters) == 0 {
				dfNRows, dfNCols := df.Dims()
				testNRows, testNCols := test.df.Dims()
				if dfNRows != testNRows || dfNCols != testNCols {
					t.Errorf("Expected dataframe dimensions (%d, %d), got (%d, %d)",
						testNRows, testNCols, dfNRows, dfNCols)
				}
			} else {
				if df.Ncol() != len(test.formatFilters) {
					t.Fatalf("Expected %d columns, got %d", len(test.formatFilters), df.Ncol())
				}
				for _, filter := range test.formatFilters {
					if !slices.Contains(df.Names(), filter.column) {
						t.Errorf("Expected column '%s' to be present", filter.column)
						continue
					}

					if filter.filterRows {
						col := df.Col(filter.column)
						for i := 0; i < col.Len(); i++ {
							if fmt.Sprintf("%v", col.Val(i)) != filter.value {
								t.Errorf("Expected column '%s' to be filtered to '%s', got %v",
									filter.column, filter.value, col.Val(i))
							}
						}
					}
				}
			}
		})
	}
}

func TestSortDataFrame(t *testing.T) {
	tests := map[string]struct {
		df         dataframe.DataFrame
		sortColumn string
		reverse    bool
		wantError  bool
	}{
		"sort ascending": {
			df:         dataframe.New(series.New([]string{"b", "a", "c"}, series.String, "col1")),
			sortColumn: "col1",
		},
		"sort descending": {
			df:         dataframe.New(series.New([]string{"b", "a", "c"}, series.String, "col1")),
			sortColumn: "col1",
			reverse:    true,
		},
		"sort int": {
			df: dataframe.New(
				series.New([]string{"b", "a", "c"}, series.String, "col1"),
				series.New([]int{2, 1, 3}, series.Int, "col2"),
			),
			sortColumn: "col2",
		},
		"sort float": {
			df:         dataframe.New(series.New([]float64{2.0, 1.0, 3.0}, series.Float, "col1")),
			sortColumn: "col1",
		},
		"lowercase sort columns": {
			df:         dataframe.New(series.New([]string{"b", "a", "c"}, series.String, "col1")),
			sortColumn: "col1",
		},
		"non-existent column": {
			df:         dataframe.New(series.New([]string{"b", "a", "c"}, series.String, "col1")),
			sortColumn: "invalid", wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			df, err := SortDataFrame(test.df, test.sortColumn, test.reverse)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error, got '%s'", err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got '%s'", err.Error())
				}

				col := df.Col(test.sortColumn)
				for i := 0; i < col.Len()-1; i++ {
					if test.reverse {
						if col.Elem(i + 1).Greater(col.Elem(i)) {
							t.Errorf("Expected column '%s' to be sorted in descending order, got %v",
								test.sortColumn, col.Records())
						}
					} else {
						if col.Elem(i + 1).Less(col.Elem(i)) {
							t.Errorf("Expected column '%s' to be sorted in ascending order, got %v",
								test.sortColumn, col.Records())
						}
					}
				}
			}
		})
	}
}

func TestDataFrameToStringData(t *testing.T) {
	tests := map[string]struct {
		df       dataframe.DataFrame
		expected [][]string
	}{
		"only headers": {
			df: dataframe.New(
				series.New([]string{}, series.String, "col1"),
				series.New([]string{}, series.String, "col2"),
			),
			expected: [][]string{},
		},
		"one column": {
			df:       dataframe.New(series.New([]string{"a", "b", "c"}, series.String, "col1")),
			expected: [][]string{{"a"}, {"b"}, {"c"}},
		},
		"multiple columns": {
			df: dataframe.New(
				series.New([]string{"a", "b", "c"}, series.String, "col1"),
				series.New([]int{1, 2, 3}, series.Int, "col2"),
				series.New([]bool{true, false, true}, series.Bool, "col3"),
			),
			expected: [][]string{{"a", "1", "true"}, {"b", "2", "false"}, {"c", "3", "true"}},
		},
		"null values": {
			df:       dataframe.New(series.New([]any{1, "b", nil, 4}, series.Int, "col1")),
			expected: [][]string{{"1"}, {"-"}, {"-"}, {"4"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := DataFrameToStringData(test.df)
			if !reflect.DeepEqual(got, test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, got)
			}
		})
	}
}
