/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func controlCommandArguments(t *testing.T) [][]string {
	t.Helper()
	return [][]string{
		{"create", "-t", "token", "-n", "analysis", "-f", writeSerialSpec(t)},
		{"run", "-t", "token", "-n", "analysis", "-f", writeSerialSpec(t)},
		{"start", "-t", "token", "-w", "analysis"},
		{"restart", "-t", "token", "-w", "analysis"},
		{
			"restart", "-t", "token", "-w", "analysis",
			"-f", writeSerialSpec(t),
		},
	}
}

func TestControlCommandsBoundResponses(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(17*1024*1024))
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)

	for _, arguments := range controlCommandArguments(t) {
		_, err := ExecuteCommand(NewRootCmd(), arguments...)
		if err == nil || !strings.Contains(err.Error(), "exceeds 16 MiB") {
			t.Fatalf(
				"%s did not bound its control response: %v",
				arguments[0],
				err,
			)
		}
	}
}

func TestControlCommandsHaveOperationTimeout(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)
	previousTimeout := controlOperationTimeout
	controlOperationTimeout = 10 * time.Millisecond
	t.Cleanup(func() { controlOperationTimeout = previousTimeout })

	for _, arguments := range controlCommandArguments(t) {
		_, err := ExecuteCommand(NewRootCmd(), arguments...)
		if err == nil {
			t.Fatalf("%s did not time out", arguments[0])
		}
	}
}
