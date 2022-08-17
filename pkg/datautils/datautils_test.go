/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package datautils

import (
	"testing"
	"time"

	"golang.org/x/exp/slices"
)

func TestHasAnyPrefix(t *testing.T) {
	tests := []struct {
		prefixes []string
		str      string
		want     bool
	}{
		{prefixes: []string{"foo"}, str: "foobar", want: true},
		{prefixes: []string{"foo"}, str: "bar", want: false},
		{prefixes: []string{"foo", "bar"}, str: "foo", want: true},
		{prefixes: []string{"foo", "bar"}, str: "foobar", want: true},
		{prefixes: []string{"foo", "bar"}, str: "baz", want: false},
		{prefixes: []string{}, str: "foobar", want: false},
		{prefixes: []string{"foo", "bar"}, str: "", want: false},
	}
	for _, test := range tests {
		got := HasAnyPrefix(test.str, test.prefixes)
		if got != test.want {
			t.Errorf("Expected %t, got %t", test.want, got)
		}
	}
}

func TestFromIsoToTimestamp(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		dateIso   string
		wantError bool
		want      time.Time
	}{
		{
			dateIso: "2020-01-01T03:16:45",
			want:    time.Date(2020, 1, 1, 3, 16, 45, 0, time.UTC),
		},
		{
			dateIso: now.Format("2006-01-02T15:04:05"),
			want:    now.Truncate(time.Second),
		},
		{
			dateIso:   "2020-01-01T00:00:00Z",
			wantError: true,
		},
		{
			dateIso:   "09:30:00Z",
			wantError: true,
		},
		{
			dateIso:   "",
			wantError: true,
		},
	}
	for _, test := range tests {
		got, err := FromIsoToTimestamp(test.dateIso)
		if test.wantError {
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
		} else {
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if got != test.want {
				t.Errorf("Expected %v, got %v", test.want, got)
			}
		}
	}
}

func TestSplitLinesNoEmpty(t *testing.T) {
	tests := []struct {
		arg  string
		want []string
	}{
		{arg: "", want: []string{}},
		{arg: "a", want: []string{"a"}},
		{arg: "a\nb", want: []string{"a", "b"}},
		{arg: "a\nb\nc", want: []string{"a", "b", "c"}},
		{arg: "a\nb\nc\n", want: []string{"a", "b", "c"}},
		{arg: "a\n\nb\n\nc", want: []string{"a", "b", "c"}},
	}
	for _, test := range tests {
		got := SplitLinesNoEmpty(test.arg)
		if !slices.Equal(got, test.want) {
			t.Errorf("Expected %v, got %v", test.want, got)
		}
	}
}
