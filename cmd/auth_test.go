/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"bytes"
	"context"
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
	"time"

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
		"--callback-port",
		strconv.Itoa(port),
	)
	if err != nil {
		t.Fatalf("browser login failed: %v, output=%s", err, out)
	}
	if !strings.Contains(
		out,
		url.QueryEscape("http://127.0.0.1:"+strconv.Itoa(port)+"/callback"),
	) {
		t.Errorf("expected the explicit callback port in %q", out)
	}
	if !strings.Contains(out, "Logged in to "+server.URL) {
		t.Errorf("expected a success message in the output, got %q", out)
	}
}

const retiredLoopbackPortEnv = "REANA_CLIENT_LOGIN_LOOPBACK_PORT"

// newLoopbackDiscoveryServer serves discovery metadata with the given raw
// JSON value of the advertised callback port; an empty value omits the field.
func newLoopbackDiscoveryServer(
	t *testing.T,
	advertisedPort string,
) *httptest.Server {
	t.Helper()
	extra := ""
	if advertisedPort != "" {
		extra = `,"reana_cli_loopback_port":` + advertisedPort
	}
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/.well-known/openid-configuration" {
				t.Errorf("unexpected request to %q", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "device_authorization_endpoint":"https://issuer.example.org/device",
                "reana_cli_client_id":"reana-cli"%s
            }`, extra)
		},
	))
	t.Cleanup(server.Close)
	trustAuthTestServer(t, server)
	return server
}

func TestLoginCommandRejectsHeadlessWithCallbackPort(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	for _, port := range []string{"0", "8899"} {
		t.Run(port, func(t *testing.T) {
			// The server URL is unreachable: the options are rejected
			// before discovery.
			_, err := ExecuteCommand(
				NewRootCmd(),
				"login",
				"--server-url",
				"https://reana.invalid",
				"--headless",
				"--callback-port",
				port,
			)
			if err == nil || !strings.Contains(
				err.Error(),
				"--headless and --callback-port cannot be used together",
			) {
				t.Fatalf("expected an option conflict, got %v", err)
			}
		})
	}
}

func TestLoginCommandRejectsInvalidCallbackPort(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	for _, port := range []string{"-1", "65536", "abc", "88.5"} {
		t.Run(port, func(t *testing.T) {
			_, err := ExecuteCommand(
				NewRootCmd(),
				"login",
				"--server-url",
				"https://reana.invalid",
				"--callback-port="+port,
			)
			if err == nil ||
				!strings.Contains(err.Error(), "callback-port") {
				t.Fatalf("expected an invalid port error, got %v", err)
			}
		})
	}
}

func TestLoginCommandStopsOnRetiredPortVariable(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv(retiredLoopbackPortEnv, "8899")
	for name, arguments := range map[string][]string{
		"no option":   {},
		"with option": {"--callback-port", "8899"},
		"with zero":   {"--callback-port", "0"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ExecuteCommand(
				NewRootCmd(),
				append(
					[]string{"login", "--server-url", "https://reana.invalid"},
					arguments...,
				)...,
			)
			if err == nil {
				t.Fatal("stale variable did not stop browser login")
			}
			for _, want := range []string{
				retiredLoopbackPortEnv + " is no longer a client input",
				"reana-client-go login --callback-port PORT",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("guidance = %v, want %q", err, want)
				}
			}
		})
	}
}

func TestLoginCommandIgnoresBlankRetiredPortVariable(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	server := newLoopbackDiscoveryServer(t, `"oops"`)
	for _, value := range []string{"", "   "} {
		t.Run(strconv.Quote(value), func(t *testing.T) {
			t.Setenv(retiredLoopbackPortEnv, value)
			_, err := ExecuteCommand(
				NewRootCmd(),
				"login",
				"--server-url",
				server.URL,
			)
			// Login continues to the port selection, which rejects the
			// invalid advertised port.
			if err == nil ||
				strings.Contains(err.Error(), "no longer a client input") ||
				!strings.Contains(err.Error(), "reana_cli_loopback_port") {
				t.Fatalf("expected an advertised port error, got %v", err)
			}
		})
	}
}

func TestRetiredPortVariableDoesNotAffectOtherCommands(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv(retiredLoopbackPortEnv, "8899")
	failing := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	))
	defer failing.Close()
	trustAuthTestServer(t, failing)

	// Device login reaches discovery instead of stopping on the variable.
	_, err := ExecuteCommand(
		NewRootCmd(),
		"login",
		"--server-url",
		failing.URL,
		"--headless",
	)
	if err == nil || !strings.Contains(err.Error(), "could not discover") {
		t.Fatalf("expected a discovery error, got %v", err)
	}

	out, err := ExecuteCommand(NewRootCmd(), "version")
	if err != nil || strings.Contains(out, retiredLoopbackPortEnv) {
		t.Fatalf("version failed: %v, output=%s", err, out)
	}
}

func TestLoginCommandCallbackPortReachesPortSelection(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv(retiredLoopbackPortEnv, "")
	// No browser opener is found, so that no real browser starts.
	t.Setenv("PATH", t.TempDir())
	server := newLoopbackDiscoveryServer(t, `"oops"`)

	t.Run("invalid advertised port without option", func(t *testing.T) {
		_, err := ExecuteCommand(
			NewRootCmd(),
			"login",
			"--server-url",
			server.URL,
		)
		if err == nil ||
			!strings.Contains(err.Error(), "reana_cli_loopback_port") ||
			!strings.Contains(err.Error(), "--callback-port") {
			t.Fatalf("expected an advertised port error, got %v", err)
		}
	})

	t.Run("occupied explicit port", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		port := listener.Addr().(*net.TCPAddr).Port
		_, err = ExecuteCommand(
			NewRootCmd(),
			"login",
			"--server-url",
			server.URL,
			"--callback-port",
			strconv.Itoa(port),
		)
		// The override bypasses the invalid advertised port.
		if err == nil ||
			!strings.Contains(err.Error(), "fixed by --callback-port") {
			t.Fatalf("expected an occupied port error, got %v", err)
		}
	})

	t.Run("explicit zero", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			500*time.Millisecond,
		)
		defer cancel()
		root := NewRootCmd()
		buffer := new(bytes.Buffer)
		root.SetOut(buffer)
		root.SetErr(buffer)
		root.SetArgs(
			[]string{
				"login",
				"--server-url",
				server.URL,
				"--callback-port",
				"0",
			},
		)
		err := root.ExecuteContext(ctx)
		// Zero forces an OS-assigned port despite the invalid advertised
		// port: login waits for the callback until the context ends.
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("expected a login timeout, got %v", err)
		}
		if !strings.Contains(
			buffer.String(),
			"redirect_uri="+url.QueryEscape("http://127.0.0.1:"),
		) {
			t.Fatalf("expected a loopback redirect URI in %q", buffer.String())
		}
	})
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
		!strings.Contains(err.Error(), "No REANA server is configured") {
		t.Fatalf("expected a no-active-cluster error, got %v", err)
	}
}
