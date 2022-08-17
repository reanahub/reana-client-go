/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package datautils gives extra functions to manipulate data structures like strings and slices.
package datautils

import (
	"strings"
	"time"
)

// HasAnyPrefix checks if the string s has any prefixes, by running strings.HasPrefix for each one.
func HasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// FromIsoToTimestamp converts a string date in the ISO format to a timestamp.
func FromIsoToTimestamp(date string) (time.Time, error) {
	timestamp, err := time.Parse("2006-01-02T15:04:05", date)

	if err != nil {
		return time.Time{}, err
	}
	return timestamp, nil
}

// SplitLinesNoEmpty splits a given string into a list where each line is a list item.
// In contrary to strings.Split, SplitLinesNoEmpty ignores empty lines.
func SplitLinesNoEmpty(str string) []string {
	splitFn := func(c rune) bool {
		return c == '\n'
	} // Ignores empty string after \n, unlike strings.Split
	lines := strings.FieldsFunc(
		str,
		splitFn,
	)
	return lines
}

// RemoveFromSlice removed the given elem from the slice.
func RemoveFromSlice[T comparable](slice []T, elem T) []T {
	for i, other := range slice {
		if other == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
