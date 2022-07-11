/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/spf13/cobra"
)

// Available run statuses
var runStatuses = []string{
	"created",
	"running",
	"finished",
	"failed",
	"deleted",
	"stopped",
	"queued",
	"pending",
}

// Files black list
var FilesBlacklist = []string{".git/", "/.git/"}

func ExecuteCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()

	return buf.String(), err
}

func DisplayJsonOutput(output any) {
	byteArray, err := json.MarshalIndent(output, "", "  ")

	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(string(byteArray))
}

func NewRequest(token string, serverURL string, endpoint string) []byte {
	// disable certificate security checks
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	url := serverURL + endpoint + "?access_token=" + token
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	return respBytes
}

func ParseListFilters(filter []string, filterNames []string) ([]string, string) {
	searchFilters := make(map[string][]string)
	var statusFilters []string

	for _, value := range filter {
		if !strings.Contains(value, "=") {
			fmt.Println("Error: Wrong input format. Please use --filter filter_name=filter_value")
			os.Exit(1)
		}

		filterNameAndValue := strings.SplitN(value, "=", 2)
		filterName := strings.ToLower(filterNameAndValue[0])
		filterValue := filterNameAndValue[1]

		if !slices.Contains(filterNames, filterName) {
			fmt.Printf("Error: Filter %s is not valid", filterName)
			os.Exit(1)
		}

		if filterName == "status" && !slices.Contains(runStatuses, filterValue) {
			fmt.Printf("Error: Input status value %s is not valid. ", filterValue)
			os.Exit(1)
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
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
		searchFiltersString = string(searchFiltersByteArray)
	}

	return statusFilters, searchFiltersString
}

func HasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
