/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRetiredExportsGiveGuidanceButAllowHelpAndVersion(t *testing.T) {
	for _, names := range [][]string{{"REANA_SERVER_URL"}, {"REANA_SERVER_TLS_VERIFY"}, {"REANA_SERVER_URL", "REANA_SERVER_TLS_VERIFY"}} {
		t.Run(strings.Join(names, "+"), func(t *testing.T) {
			t.Cleanup(viper.Reset)
			for _, name := range names {
				t.Setenv(name, "stale")
			}
			_, err := ExecuteCommand(NewRootCmd(), "ping", "-t", "a.b.c")
			if err == nil ||
				!strings.Contains(err.Error(), "Unset them, then run") {
				t.Fatalf("migration guidance missing: %v", err)
			}
			for _, name := range names {
				if !strings.Contains(err.Error(), name) {
					t.Fatalf("missing variable: %v", err)
				}
			}
			for _, arguments := range [][]string{{"--help"}, {"help"}, {"help", "ping"}, {"login", "--help"}, {"version"}, {"completion", "bash"}} {
				if _, err := ExecuteCommand(NewRootCmd(), arguments...); err != nil {
					t.Fatalf("help blocked: %v", err)
				}
			}
		})
	}
}

func TestConflictingLoginFlags(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/client.json")
	for _, flags := range [][]string{
		{"--server", "https://first.example.org", "--server-url", "https://second.example.org"},
		{"--tls-verify", "--no-tls-verify"},
		{"--tls-verify=false", "--no-tls-verify"},
	} {
		_, err := ExecuteCommand(
			NewRootCmd(),
			append([]string{"login"}, flags...)...)
		if err == nil {
			t.Fatal("conflicting flags accepted")
		}
	}
}

func TestCommandsReportUntrustedServer(t *testing.T) {
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("untrusted server received a request")
		}),
	)
	defer server.Close()
	for _, command := range []string{"ping", "list", "info"} {
		t.Run(command, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			savedTestServer(t, server.URL, true)
			output, err := ExecuteCommand(
				NewRootCmd(),
				command,
				"-t",
				"synthetic.jwt.token",
			)
			if err == nil ||
				!strings.Contains(
					err.Error(),
					"TLS certificate is not trusted",
				) ||
				!strings.Contains(
					err.Error(),
					"login --server "+server.URL+" --no-tls-verify",
				) {
				t.Fatalf("missing certificate guidance: %s, %v", output, err)
			}
			if strings.Contains(output, "Authenticated as:") ||
				strings.Contains(err.Error(), "synthetic.jwt.token") {
				t.Fatalf("misleading or sensitive output: %s, %v", output, err)
			}
		})
	}
}

func TestFirstLoginSavesBypassAndReloginInheritsIt(t *testing.T) {
	for _, flag := range []string{"--server", "--server-url"} {
		t.Run(flag, func(t *testing.T) {
			t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/client.json")
			t.Setenv("REANA_SERVER_CA_CERTS", "")
			viper.Reset()
			t.Cleanup(viper.Reset)
			var server *httptest.Server
			server = httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/api/.well-known/openid-configuration":
						fmt.Fprintf(
							w,
							`{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"device_authorization_endpoint":%q,"reana_cli_client_id":"cli"}`,
							server.URL,
							server.URL+"/auth",
							server.URL+"/token",
							server.URL+"/device",
						)
					case "/device":
						fmt.Fprintf(
							w,
							`{"device_code":"device","user_code":"ABCD","verification_uri":%q,"expires_in":60,"interval":1}`,
							server.URL+"/verify",
						)
					case "/token":
						fmt.Fprint(
							w,
							`{"access_token":"a.b.c","refresh_token":"refresh","expires_in":3600}`,
						)
					default:
						fmt.Fprint(
							w,
							`{"email":"user@example.org","reana_server_version":"0.95"}`,
						)
					}
				}),
			)
			defer server.Close()
			output, err := ExecuteCommand(
				NewRootCmd(),
				"login",
				"--headless",
				flag,
				server.URL,
				"--no-tls-verify",
			)
			if err != nil ||
				!strings.Contains(output, "TLS verification: disabled") {
				t.Fatalf("login = %s, %v", output, err)
			}
			store, _ := auth.NewStore()
			entry, err := store.Get(server.URL)
			if err != nil || entry.TLS == nil || *entry.TLS.Verify {
				t.Fatalf("policy not saved: %+v, %v", entry, err)
			}
			if _, err := ExecuteCommand(NewRootCmd(), "login", "--headless"); err != nil {
				t.Fatalf("re-login lost saved settings: %v", err)
			}
			if output, err := ExecuteCommand(NewRootCmd(), "ping"); err != nil ||
				!strings.Contains(output, "TLS verification: disabled") {
				t.Fatalf("saved ping = %s, %v", output, err)
			}
			trustAuthTestServer(t, server)
			if output, err := ExecuteCommand(NewRootCmd(), "ping"); err != nil ||
				!strings.Contains(output, "TLS verification: enabled") {
				t.Fatalf("CA ping = %s, %v", output, err)
			}
			if _, err := ExecuteCommand(NewRootCmd(), "login", "--headless", "--tls-verify"); err != nil {
				t.Fatal(err)
			}
			entry, _ = store.Get(server.URL)
			if !*entry.TLS.Verify {
				t.Fatal("positive flag did not restore verification")
			}
		})
	}
}

func TestPingKeepsTLSPolicyUntilNextInvocation(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(
				w,
				`{"email":"user@example.org","reana_server_version":"0.95"}`,
			)
		}),
	)
	defer server.Close()
	savedTestServer(t, server.URL, false)
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	root := NewRootCmd()
	ping, _, err := root.Find([]string{"ping"})
	if err != nil {
		t.Fatal(err)
	}
	run := ping.RunE
	ping.RunE = func(cmd *cobra.Command, args []string) error {
		// Model another terminal updating the saved choice after validation.
		verify := true
		if _, err := store.Put(server.URL, auth.Credentials{TLS: &auth.TLSSettings{Verify: &verify}}, false); err != nil {
			return err
		}
		return run(cmd, args)
	}
	output, err := ExecuteCommand(root, "ping", "-t", "a.b.c")
	if err != nil || !strings.Contains(output, "TLS verification: disabled") {
		t.Fatalf(
			"invocation did not retain its TLS policy: %s, %v",
			output,
			err,
		)
	}
	_, err = ExecuteCommand(NewRootCmd(), "ping", "-t", "a.b.c")
	if err == nil ||
		!strings.Contains(err.Error(), "certificate is not trusted") {
		t.Fatalf("next invocation did not read the new policy: %v", err)
	}
}
