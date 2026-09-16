/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/reanahub/reana-client-go/pkg/auth"
)

func trustAuthTestServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	certificate := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(caPath, certificate, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REANA_SERVER_CA_CERTS", caPath)
}

func TestOpenBrowserReturnsWrappedErrorWhenNoOpenerIsFound(t *testing.T) {
	t.Setenv("PATH", "")
	err := openBrowser("https://reana.example.org")
	if err == nil ||
		!strings.Contains(err.Error(), "could not find browser opener") {
		t.Fatalf(
			"openBrowser error = %v, want wrapped opener-not-found error",
			err,
		)
	}
}

func TestLoginCommandRejectsInvalidServerURL(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")

	_, err := ExecuteCommand(
		NewRootCmd(),
		"login",
		"--server-url",
		"http://reana.example.org",
	)
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected an HTTPS-required error, got %v", err)
	}
}

func TestLoginCommandPropagatesDiscoveryError(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	))
	defer server.Close()
	trustAuthTestServer(t, server)

	_, err := ExecuteCommand(
		NewRootCmd(),
		"login",
		"--server-url",
		server.URL,
	)
	if err == nil || !strings.Contains(err.Error(), "could not discover") {
		t.Fatalf("expected a discovery error, got %v", err)
	}
}

func TestLoginDeviceCommandCompletesFullFlow(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/.well-known/openid-configuration":
				fmt.Fprintf(w, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":%q,
                "device_authorization_endpoint":%q,
                "reana_cli_client_id":"reana-cli"
            }`, server.URL+"/token", server.URL+"/device")
			case "/device":
				fmt.Fprint(
					w,
					`{"device_code":"device","user_code":"ABCD","verification_uri":"https://issuer.example.org/verify","expires_in":300,"interval":1}`,
				)
			case "/token":
				fmt.Fprint(
					w,
					`{"access_token":"a.b.c","refresh_token":"refresh","expires_in":3600}`,
				)
			default:
				t.Errorf("unexpected request to %q", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		},
	))
	defer server.Close()
	trustAuthTestServer(t, server)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"login",
		"--server-url",
		server.URL,
		"--headless",
	)
	if err != nil {
		t.Fatalf("device login failed: %v, output=%s", err, out)
	}
	if !strings.Contains(out, "ABCD") {
		t.Errorf("expected the user code in the output, got %q", out)
	}
	if !strings.Contains(out, "Logged in to "+server.URL) {
		t.Errorf("expected a success message in the output, got %q", out)
	}
}

// TestLoginBrowserCommandCompletesFullFlow drives newLoginCmd's real,
// non-injectable openBrowser through a throwaway "xdg-open" script placed
// first on PATH; the script uses curl to follow the fake IdP's redirect back
// to the loopback callback, exactly like a real browser would.
func TestLoginBrowserCommandCompletesFullFlow(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test supplies a fake xdg-open implementation")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl not available")
	}
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()
	t.Setenv("REANA_CLIENT_LOGIN_LOOPBACK_PORT", strconv.Itoa(port))

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/.well-known/openid-configuration":
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":%q,
                "token_endpoint":%q,
                "reana_cli_client_id":"reana-cli"
            }`, server.URL+"/auth", server.URL+"/token")
			case "/auth":
				query := r.URL.Query()
				redirect := query.Get("redirect_uri") +
					"?code=test-code&state=" + url.QueryEscape(query.Get("state"))
				http.Redirect(w, r, redirect, http.StatusFound)
			case "/token":
				contents, _ := io.ReadAll(r.Body)
				form, _ := url.ParseQuery(string(contents))
				if form.Get("grant_type") != "authorization_code" ||
					form.Get("code") != "test-code" {
					t.Errorf("unexpected token exchange form: %v", form)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(
					w,
					`{"access_token":"a.b.c","refresh_token":"refresh","expires_in":3600}`,
				)
			default:
				t.Errorf("unexpected request to %q", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		},
	))
	defer server.Close()
	trustAuthTestServer(t, server)

	scriptDir := t.TempDir()
	script := "#!/bin/sh\ncurl -k -s -o /dev/null -L \"$1\"\n"
	if err := os.WriteFile(
		filepath.Join(scriptDir, "xdg-open"),
		[]byte(script),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv(
		"PATH",
		scriptDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)

	out, err := ExecuteCommand(
		NewRootCmd(),
		"login",
		"--server-url",
		server.URL,
	)
	if err != nil {
		t.Fatalf("browser login failed: %v, output=%s", err, out)
	}
	if !strings.Contains(out, "Logged in to "+server.URL) {
		t.Errorf("expected a success message in the output, got %q", out)
	}
}

func TestLogoutCommandRevokesAndClearsStoredCredentials(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/revoke" {
				t.Errorf("unexpected request to %q", r.URL.Path)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	))
	defer server.Close()
	trustAuthTestServer(t, server)

	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(server.URL, auth.Credentials{
		ClientID:           "reana-cli",
		RevocationEndpoint: server.URL + "/revoke",
		AccessToken:        "access",
		RefreshToken:       "refresh",
	}, true); err != nil {
		t.Fatal(err)
	}

	out, err := ExecuteCommand(NewRootCmd(), "logout")
	if err != nil {
		t.Fatalf("logout failed: %v, output=%s", err, out)
	}
	if !strings.Contains(out, "Logged out from "+server.URL) {
		t.Errorf("expected a success message in the output, got %q", out)
	}
	stored, err := store.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AccessToken != "" || stored.RefreshToken != "" {
		t.Errorf("expected tokens to be cleared, got %+v", stored)
	}
}

func TestLogoutCommandErrorsWhenNotLoggedIn(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")

	_, err := ExecuteCommand(NewRootCmd(), "logout")
	if err == nil ||
		!strings.Contains(err.Error(), "not connected to any REANA cluster") {
		t.Fatalf("expected a no-active-cluster error, got %v", err)
	}
}
