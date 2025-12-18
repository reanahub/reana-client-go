/*
This file is part of REANA.
Copyright (C) 2022, 2023 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package formatter

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"golang.org/x/exp/slices"
)

func TestParseFormatParameters(t *testing.T) {
	tests := map[string]struct {
		formatOptions    []string
		filterRows       bool
		wantFilteredRows bool
	}{
		"no params": {
			formatOptions: []string{},
			filterRows:    false, wantFilteredRows: false,
		},
		"without filtering": {
			formatOptions: []string{"column", "column2"},
			filterRows:    false, wantFilteredRows: false,
		},
		"with filterRows and no filters": {
			formatOptions: []string{"column"},
			filterRows:    true, wantFilteredRows: false,
		},
		"with filters and filterRows": {
			formatOptions: []string{"column=value"},
			filterRows:    true, wantFilteredRows: true,
		},
		"with filters and no filterRows": {
			formatOptions: []string{"column=value"},
			filterRows:    false, wantFilteredRows: false,
		},
		"multiple filters": {
			formatOptions: []string{"column=value", "column2=value2"},
			filterRows:    true, wantFilteredRows: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters := ParseFormatParameters(
				test.formatOptions,
				test.filterRows,
			)
			if len(filters) != len(test.formatOptions) {
				t.Errorf(
					"Expected %d filters, got %d",
					len(test.formatOptions),
					len(filters),
				)
			}
			for i, filter := range filters {
				filterCol := strings.SplitN(test.formatOptions[i], "=", 2)[0]
				if filter.column != filterCol {
					t.Errorf(
						"Expected filter column %s, got %s",
						test.formatOptions[i],
						filter.column,
					)
				}

				if test.wantFilteredRows {
					if !filter.filterRows {
						t.Errorf("Expected filterRows to be true, got false")
					}
					if filter.value == "" {
						t.Errorf("Expected a filter value, got empty string")
					}
				} else {
					if filter.filterRows {
						t.Errorf("Expected filterRows to be false, got true")
					}
					if filter.value != "" {
						t.Errorf("Expected empty filter value, got %s", filter.value)
					}
				}
			}
		})
	}
}

func TestFormatDataFrame(t *testing.T) {
	tests := map[string]struct {
		df            dataframe.DataFrame
		formatFilters []FormatFilter
		wantError     bool
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
			formatFilters: []FormatFilter{
				{column: "col2", filterRows: true, value: "2"},
			},
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
		"invalid format column": {
			df: dataframe.New(
				series.New([]string{"a", "b"}, series.String, "col1"),
				series.New([]int{1, 2}, series.Int, "col2"),
			),
			formatFilters: []FormatFilter{{column: "col3"}},
			wantError:     true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			df, err := FormatDataFrame(test.df, test.formatFilters)
			if test.wantError {
				if err == nil {
					t.Fatalf("wanted error, got nil")
				}
			} else {
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
			}
		})
	}
}

func TestSortDataFrame(t *testing.T) {
	tests := map[string]struct {
		df         dataframe.DataFrame
		expected   dataframe.DataFrame
		sortColumn string
		reverse    bool
		readToRaw  map[string]int64
		humanRead  bool
		wantError  bool
	}{
		"sort ascending": {
			df: dataframe.New(
				series.New([]string{"b", "a", "c"}, series.String, "col1"),
			),
			expected: dataframe.New(
				series.New([]string{"a", "b", "c"}, series.String, "col1"),
			),
			sortColumn: "col1",
		},
		"sort descending": {
			df: dataframe.New(
				series.New([]string{"b", "a", "c"}, series.String, "col1"),
			),
			expected: dataframe.New(
				series.New([]string{"c", "b", "a"}, series.String, "col1"),
			),
			sortColumn: "col1",
			reverse:    true,
		},
		"sort int": {
			df: dataframe.New(
				series.New([]string{"b", "a", "c"}, series.String, "col1"),
				series.New([]int{2, 1, 3}, series.Int, "col2"),
			),
			expected: dataframe.New(
				series.New([]int{1, 2, 3}, series.Int, "col2"),
			),
			sortColumn: "col2",
		},
		"sort float": {
			df: dataframe.New(
				series.New([]float64{2.0, 1.0, 3.0}, series.Float, "col1"),
			),
			expected: dataframe.New(
				series.New([]float64{1.0, 2.0, 3.0}, series.Float, "col1"),
			),
			sortColumn: "col1",
		},
		"sort run_numbers": {
			df: dataframe.New(
				series.New(
					[]string{"1", "2.2", "10", "9.1", "1.15", "2.10"},
					series.String,
					"run_number",
				),
			),
			expected: dataframe.New(
				series.New(
					[]string{"1", "1.15", "2.2", "2.10", "9.1", "10"},
					series.String,
					"run_number",
				),
			),
			sortColumn: "run_number",
		},
		"sort size human_readable": {
			df: dataframe.New(
				series.New(
					[]string{"255 KiB", "1.92 MiB", "192 KiB", "1.1 GiB"},
					series.String,
					"size",
				),
			),
			expected: dataframe.New(
				series.New(
					[]string{"192 KiB", "255 KiB", "1.92 MiB", "1.1 GiB"},
					series.String,
					"size",
				),
			),
			sortColumn: "size",
			readToRaw: map[string]int64{
				"255 KiB":  261120,
				"1.92 MiB": 2013265,
				"192 KiB":  196608,
				"1.1 GiB":  1181116006,
			},
			humanRead: true,
		},
		"non-existent column": {
			df: dataframe.New(
				series.New([]string{"b", "a", "c"}, series.String, "col1"),
			),
			sortColumn: "invalid", wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			df, err := SortDataFrame(
				test.df,
				test.sortColumn,
				test.reverse,
				test.readToRaw,
				test.humanRead,
			)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error, got '%s'", err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got '%s'", err.Error())
				}

				col := df.Col(test.sortColumn)
				expectedCol := test.expected.Col(test.sortColumn)
				if !reflect.DeepEqual(col, expectedCol) {
					t.Errorf("The given dataframe and the expected one do not match!")
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
			df: dataframe.New(
				series.New([]string{"a", "b", "c"}, series.String, "col1"),
			),
			expected: [][]string{{"a"}, {"b"}, {"c"}},
		},
		"multiple columns": {
			df: dataframe.New(
				series.New([]string{"a", "b", "c"}, series.String, "col1"),
				series.New([]int{1, 2, 3}, series.Int, "col2"),
				series.New([]bool{true, false, true}, series.Bool, "col3"),
			),
			expected: [][]string{
				{"a", "1", "true"},
				{"b", "2", "false"},
				{"c", "3", "true"},
			},
		},
		"null values": {
			df: dataframe.New(
				series.New([]any{1, "b", nil, 4}, series.Int, "col1"),
			),
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

func TestFormatSessionURI(t *testing.T) {
	tests := map[string]struct {
		serverURL string
		path      string
		token     string
		want      string
	}{
		"regular uri": {
			serverURL: "https://server.com",
			path:      "/api/",
			token:     "token",
			want:      "https://server.com/api/?token=token",
		},
		"no path": {
			serverURL: "https://server.com/",
			path:      "",
			token:     "token",
			want:      "https://server.com/?token=token",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := FormatSessionURI(test.serverURL, test.path, test.token)
			if got != test.want {
				t.Errorf("Expected %s, got %s", test.want, got)
			}
		})
	}
}
