/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

func TestNewFilters(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		inputFilters     []string
		testFilter       bool
		wantError        bool
	}{
		"empty filters": {
			singleFilterKeys: []string{},
			multiFilterKeys:  []string{},
		},
		"no filter values": {
			singleFilterKeys: []string{"single"},
			multiFilterKeys:  []string{"multi"},
		},
		"with filter value": {
			singleFilterKeys: []string{"single"},
			multiFilterKeys:  []string{"multi"},
			inputFilters:     []string{"single=value"},
			testFilter:       true,
		},
		"invalid filter input": {
			singleFilterKeys: []string{"single"},
			multiFilterKeys:  []string{"multi"},
			inputFilters:     []string{"invalid_input"},
			wantError:        true,
		},
		"invalid filter key": {
			singleFilterKeys: []string{"single"},
			multiFilterKeys:  []string{"multi"},
			inputFilters:     []string{"key=value"},
			wantError:        true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(
				test.singleFilterKeys,
				test.multiFilterKeys,
				test.inputFilters,
			)
			if test.wantError {
				if err == nil {
					t.Errorf(
						"Expected error for NewFilters(%v, %v, %v), got nil",
						test.singleFilterKeys, test.multiFilterKeys, test.inputFilters,
					)
				}
			} else if err != nil {
				t.Errorf(
					"Unexpected error for NewFilters(%v, %v, %v): '%s'",
					test.singleFilterKeys, test.multiFilterKeys, test.inputFilters, err.Error(),
				)
			} else {
				for _, key := range test.singleFilterKeys {
					if !slices.Contains(filters.SingleFilterKeys, key) {
						t.Errorf("Expected '%s' to be in %v", key, filters.SingleFilterKeys)
					}
				}
				for _, key := range test.multiFilterKeys {
					if !slices.Contains(filters.MultiFilterKeys, key) {
						t.Errorf("Expected '%s' to be in %v", key, filters.MultiFilterKeys)
					}
				}

				if test.testFilter {
					filter, err := filters.GetSingle(test.singleFilterKeys[0])
					if err != nil {
						t.Errorf("Unexpected error when getting filter: '%s'", err.Error())
					}
					if filter != "value" {
						t.Errorf("Expected filter value to be 'value', got '%s'", filter)
					}
				}
			}
		})
	}
}

func TestFiltersAddFilters(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		inputFilters     []string
		wantError        bool
	}{
		"empty filters": {
			inputFilters: []string{},
		},
		"valid filters": {
			singleFilterKeys: []string{"single"},
			multiFilterKeys:  []string{"multi"},
			inputFilters:     []string{"single=value", "multi=value"},
		},
		"invalid keys": {
			singleFilterKeys: []string{"single"},
			inputFilters:     []string{"single=value", "multi=value"},
			wantError:        true,
		},
		"invalid filter input": {
			singleFilterKeys: []string{""},
			inputFilters:     []string{"invalid_input"},
			wantError:        true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(test.singleFilterKeys, test.multiFilterKeys, []string{})
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			err = filters.AddFilters(test.inputFilters)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error for AddFilters(%v), got nil", test.inputFilters)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for AddFilters(%v): '%s'", test.inputFilters, err.Error())
			}
		})
	}
}

func TestFiltersAddFilter(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		filterKey        string
		filterValue      string
		wantError        bool
	}{
		"single value": {
			singleFilterKeys: []string{"single"},
			filterKey:        "single", filterValue: "value",
		},
		"multi value": {
			singleFilterKeys: []string{"single"}, multiFilterKeys: []string{"multi"},
			filterKey: "multi", filterValue: "value",
		},
		"invalid key": {
			multiFilterKeys: []string{"multi"}, filterKey: "single",
			filterValue: "value", wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(test.singleFilterKeys, test.multiFilterKeys, []string{})
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			err = filters.AddFilter(test.filterKey + "=" + test.filterValue)
			if test.wantError {
				if err == nil {
					t.Errorf(
						"Expected error for AddFilter(%s=%s), got nil",
						test.filterKey,
						test.filterValue,
					)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for AddFilter(%s=%s): '%s'", test.filterKey, test.filterValue, err.Error())
			} else if slices.Contains(filters.SingleFilterKeys, test.filterKey) {
				value, _ := filters.GetSingle(test.filterKey)
				if value != test.filterValue {
					t.Errorf("Expected filter value to be '%s', got '%s'", test.filterValue, value)
				}
			} else {
				values, _ := filters.GetMulti(test.filterKey)
				if !slices.Contains(values, test.filterValue) {
					t.Errorf("Expected filter values to contain '%s', got %v", test.filterValue, values)
				}
			}
		})
	}

	t.Run("invalid filter input", func(t *testing.T) {
		filters, _ := NewFilters([]string{"single"}, []string{}, []string{})
		err := filters.AddFilter("invalid_filter")
		if err == nil {
			t.Errorf("Expected error for AddFilter(%s), got nil", "invalid_filter")
		}
	})
}

func TestFiltersGetSingle(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		inputFilters     []string
		key              string
		expected         string
		wantError        bool
	}{
		"filter with value": {
			singleFilterKeys: []string{
				"single",
				"single2",
			}, inputFilters: []string{"single=value", "single2=value2"},
			key: "single", expected: "value",
		},
		"filter without value": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{},
			key: "single", expected: "",
		},
		"invalid key": {
			key: "single", wantError: true,
		},
		"wrong filter type": {
			multiFilterKeys: []string{"multi"}, inputFilters: []string{"multi=value"},
			key: "multi", wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(
				test.singleFilterKeys,
				test.multiFilterKeys,
				test.inputFilters,
			)
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			value, err := filters.GetSingle(test.key)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error for GetSingle(%s), got nil", test.key)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for GetSingle(%s): '%s'", test.key, err.Error())
			} else if value != test.expected {
				t.Errorf("Expected filter value to be '%s', got '%s'", test.expected, value)
			}
		})
	}
}

func TestFiltersGetMulti(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		inputFilters     []string
		key              string
		expected         []string
		wantError        bool
	}{
		"filter with one value": {
			multiFilterKeys: []string{
				"multi",
				"multi2",
			}, inputFilters: []string{"multi=value", "multi2=value2"},
			key: "multi", expected: []string{"value"},
		},
		"filter with multiple values": {
			multiFilterKeys: []string{
				"multi",
			}, inputFilters: []string{"multi=value", "multi=value2"},
			key: "multi", expected: []string{"value", "value2"},
		},
		"filter with no values": {
			multiFilterKeys: []string{"multi"}, inputFilters: []string{},
			key: "multi", expected: []string{},
		},
		"invalid key": {
			key: "multi", wantError: true,
		},
		"wrong filter type": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{"single=value"},
			key: "single", wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(
				test.singleFilterKeys,
				test.multiFilterKeys,
				test.inputFilters,
			)
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			values, err := filters.GetMulti(test.key)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error for GetMulti(%s), got nil", test.key)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for GetMulti(%s): '%s'", test.key, err.Error())
			} else if !slices.Equal(values, test.expected) {
				t.Errorf("Expected filter values to be %v, got %v", test.expected, values)
			}
		})
	}
}

func TestFiltersGetJson(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		inputFilters     []string
		keys             []string
		expected         string
		wantError        bool
	}{
		"empty filters": {
			keys: []string{}, expected: "",
		},
		"single value filter": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{"single=value"},
			keys: []string{"single"}, expected: `{"single":"value"}`,
		},
		"multiple value filter": {
			multiFilterKeys: []string{"multi"}, inputFilters: []string{"multi=value"},
			keys: []string{"multi"}, expected: `{"multi":["value"]}`,
		},
		"filter without value": {
			singleFilterKeys: []string{"single", "single2"}, inputFilters: []string{"single=value"},
			keys: []string{"single", "single2"}, expected: `{"single":"value"}`,
		},
		"multiple filters": {
			singleFilterKeys: []string{"single"}, multiFilterKeys: []string{"multi"},
			inputFilters: []string{
				"single=value",
				"multi=value2",
			}, keys: []string{"single", "multi"},
			expected: `{"multi":["value2"],"single":"value"}`,
		},
		"invalid key": {
			keys: []string{"key"}, wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(
				test.singleFilterKeys,
				test.multiFilterKeys,
				test.inputFilters,
			)
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			json, err := filters.GetJson(test.keys)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error for GetJson(%v), got nil", test.keys)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for GetJson(%v): '%s'", test.keys, err.Error())
			} else if json != test.expected {
				t.Errorf("Expected result to be %s, got %s", test.expected, json)
			}
		})
	}
}

func TestFiltersValidateValues(t *testing.T) {
	tests := map[string]struct {
		singleFilterKeys []string
		multiFilterKeys  []string
		inputFilters     []string
		key              string
		possibleValues   []string
		wantError        bool
	}{
		"single filter": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{"single=value"},
			key: "single", possibleValues: []string{"value"},
		},
		"multi filter": {
			multiFilterKeys: []string{"multi"}, inputFilters: []string{"multi=value"},
			key: "multi", possibleValues: []string{"value"},
		},
		"single filter with multiple options": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{"single=value2"},
			key: "single", possibleValues: []string{"value", "value2", "value3"},
		},
		"multi filter with multiple options": {
			multiFilterKeys: []string{
				"multi",
			}, inputFilters: []string{"multi=value", "multi=value2"},
			key: "multi", possibleValues: []string{"value", "value2", "value3"},
		},
		"empty single filter": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{},
			key: "single", possibleValues: []string{"value"},
		},
		"empty multi filter": {
			multiFilterKeys: []string{"multi"}, inputFilters: []string{},
			key: "multi", possibleValues: []string{"value"},
		},
		"invalid key": {
			key: "key", wantError: true,
		},
		"single filter invalid value": {
			singleFilterKeys: []string{"single"}, inputFilters: []string{"single=value"},
			key: "single", possibleValues: []string{"value2"}, wantError: true,
		},
		"multi filter invalid value": {
			multiFilterKeys: []string{
				"multi",
			}, inputFilters: []string{"multi=value", "multi=value2"},
			key: "multi", possibleValues: []string{"value2"}, wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters(
				test.singleFilterKeys,
				test.multiFilterKeys,
				test.inputFilters,
			)
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			err = filters.ValidateValues(test.key, test.possibleValues)
			if test.wantError {
				if err == nil {
					t.Errorf(
						"Expected error for ValidateValues(%s, %v), got nil",
						test.key,
						test.possibleValues,
					)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for ValidateValues(%s, %v): '%s'", test.key, test.possibleValues, err.Error())
			}
		})
	}
}

func TestFiltersGetKeyAndValue(t *testing.T) {
	tests := map[string]struct {
		filter    string
		name      string
		value     string
		wantError bool
	}{
		"regular filter":      {filter: "key=value", name: "key", value: "value"},
		"missing value":       {filter: "key=", name: "key", value: ""},
		"missing key":         {filter: "=value", name: "", value: "value"},
		"uppercase key":       {filter: "KEY=value", name: "key", value: "value"},
		"value including '='": {filter: "key=value=value", name: "key", value: "value=value"},
		"invalid input":       {filter: "invalid", wantError: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filters, err := NewFilters([]string{}, []string{}, []string{})
			if err != nil {
				t.Fatalf("Unexpected error when creating filters: '%s'", err.Error())
			}

			name, value, err := filters.getKeyAndValue(test.filter)
			if test.wantError {
				if err == nil {
					t.Errorf("Expected error for GetKeyAndValue(%s), got nil", test.filter)
				}
			} else if err != nil {
				t.Errorf("Unexpected error for GetKeyAndValue(%s): '%s'", test.filter, err.Error())
			} else if name != test.name || value != test.value {
				t.Errorf("Expected result to be %s,%s, got %s,%s", test.name, test.value, name, value)
			}
		})
	}
}

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
			filters := ParseFormatParameters(test.formatOptions, test.filterRows)
			if len(filters) != len(test.formatOptions) {
				t.Errorf("Expected %d filters, got %d", len(test.formatOptions), len(filters))
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
