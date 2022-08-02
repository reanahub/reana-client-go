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
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var FilesBlacklist = []string{".git/", "/.git/"}
var InteractiveSessionTypes = []string{"jupyter"}
var ReanaComputeBackends = map[string]string{
	"kubernetes": "Kubernetes",
	"htcondor":   "HTCondor",
	"slurm":      "Slurm",
}

func GetRunStatuses(includeDeleted bool) []string {
	runStatuses := []string{
		"created",
		"running",
		"finished",
		"failed",
		"stopped",
		"queued",
		"pending",
	}
	if includeDeleted {
		runStatuses = append(runStatuses, "deleted")
	}
	return runStatuses
}

func ExecuteCommand(cmd *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()

	return buf.String(), err
}

func NewRequest(token string, serverURL string, endpoint string) ([]byte, error) {
	// disable certificate security checks
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	url := serverURL + endpoint + "?access_token=" + token
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBytes, nil
}

func HasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func FromIsoToTimestamp(date string) (time.Time, error) {
	timestamp, err := time.Parse("2006-01-02T15:04:05", date)
	if err != nil {
		return time.Time{}, err
	}
	return timestamp, nil
}

func GetWorkflowNameAndRunNumber(workflowName string) (string, string) {
	workflowNameAndRunNumber := strings.SplitN(workflowName, ".", 2)
	if len(workflowNameAndRunNumber) < 2 {
		return workflowName, ""
	}
	return workflowNameAndRunNumber[0], workflowNameAndRunNumber[1]
}

func FormatSessionURI(serverURL string, path string, token string) string {
	return serverURL + path + "?token=" + token
}

func LogCmdFlags(cmd *cobra.Command) {
	log.Debugf("command: %s", cmd.CalledAs())
	cmd.Flags().Visit(func(f *pflag.Flag) {
		log.Debugf("%s: %s", f.Name, f.Value)
	})
}
