/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientTLSVerificationValues(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()
	t.Setenv(caCertsEnv, "")

	cases := []struct {
		value  string
		verify bool
	}{
		{"", true},
		{"   ", true},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"Yes", true},
		{"on", true},
		{"  true  ", true},
		{"0", false},
		{"false", false},
		{"FALSE", false},
		{"no", false},
		{"No", false},
		{"off", false},
		{"  off  ", false},
	}
	for _, testCase := range cases {
		t.Run(testCase.value, func(t *testing.T) {
			t.Setenv(tlsVerifyEnv, testCase.value)
			client, err := NewHTTPClient()
			if err != nil {
				t.Fatal(err)
			}
			defer client.CloseIdleConnections()
			response, err := client.Get(server.URL)
			if response != nil {
				response.Body.Close()
			}
			if testCase.verify {
				if err == nil || !strings.Contains(err.Error(), "certificate") {
					t.Fatalf("expected certificate rejection, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("explicit verification bypass failed: %v", err)
			}
		})
	}
}

func TestStrictHTTPClientIgnoresServerTLSVerify(t *testing.T) {
	output := captureTLSWarning(t)
	t.Setenv(caCertsEnv, "")
	for _, value := range []string{"false", "banana"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(tlsVerifyEnv, value)
			client, err := NewStrictHTTPClient()
			if err != nil {
				t.Fatal(err)
			}
			transport := client.Transport.(*http.Transport)
			if transport.TLSClientConfig.InsecureSkipVerify {
				t.Fatal("identity-provider certificate verification disabled")
			}
		})
	}
	if output.Len() != 0 {
		t.Fatalf("strict client emitted a bypass warning: %q", output.String())
	}
}

func TestManagerScopesTLSVerification(t *testing.T) {
	var issuerRequests atomic.Int32
	issuer := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			issuerRequests.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access"}`))
		},
	))
	defer issuer.Close()
	metadata := Metadata{
		Issuer:                      issuer.URL,
		AuthorizationEndpoint:       issuer.URL + "/authorize",
		TokenEndpoint:               issuer.URL + "/token",
		DeviceAuthorizationEndpoint: issuer.URL + "/device",
		RevocationEndpoint:          issuer.URL + "/revoke",
		CLIClientID:                 "reana-client",
	}
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(metadata)
		},
	))
	defer server.Close()
	caPath := t.TempDir() + "/ca.pem"
	var bundle []byte
	for _, peer := range []*httptest.Server{server, issuer} {
		bundle = append(bundle, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: peer.Certificate().Raw,
		})...)
	}
	if err := os.WriteFile(caPath, bundle, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(tlsVerifyEnv, "false")
	t.Setenv(configPathEnv, t.TempDir()+"/credentials.json")
	for _, trusted := range []bool{false, true} {
		name := "untrusted issuer"
		if trusted {
			name = "trusted CA bundle"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv(caCertsEnv, "")
			if trusted {
				t.Setenv(caCertsEnv, caPath)
			}
			issuerRequests.Store(0)
			manager, err := NewManager()
			if err != nil {
				t.Fatal(err)
			}
			defer manager.HTTPClient.CloseIdleConnections()
			defer manager.IdPHTTPClient.CloseIdleConnections()
			discovered, err := manager.Discover(
				context.Background(),
				server.URL,
			)
			if err != nil {
				t.Fatalf("REANA discovery failed: %v", err)
			}
			for _, endpoint := range []string{
				discovered.TokenEndpoint,
				discovered.DeviceAuthorizationEndpoint,
			} {
				var tokens tokenResponse
				_, err = manager.postForm(
					context.Background(), server.URL, endpoint, "token request",
					url.Values{"client_id": {metadata.CLIClientID}}, &tokens,
				)
				if trusted && err != nil {
					t.Fatalf(
						"trusted identity-provider request failed: %v",
						err,
					)
				}
				if !trusted &&
					(err == nil || !strings.Contains(err.Error(), "certificate")) {
					t.Fatalf(
						"expected identity-provider certificate rejection: %v",
						err,
					)
				}
			}
			warning := manager.revokeBestEffort(
				context.Background(), server.URL, discovered, "refresh-token",
			)
			if trusted {
				if warning != "" || issuerRequests.Load() != 3 {
					t.Fatalf(
						"trusted issuer: requests=%d warning=%q",
						issuerRequests.Load(),
						warning,
					)
				}
				for _, client := range []*http.Client{manager.HTTPClient, manager.IdPHTTPClient} {
					transport := client.Transport.(*http.Transport).Clone()
					transport.TLSClientConfig.ServerName = "wrong.example.org"
					hostnameClient := &http.Client{Transport: transport}
					defer hostnameClient.CloseIdleConnections()
					response, err := hostnameClient.Get(issuer.URL)
					if response != nil {
						response.Body.Close()
					}
					if err == nil ||
						!strings.Contains(err.Error(), "certificate") {
						t.Fatalf(
							"CA bundle bypassed hostname verification: %v",
							err,
						)
					}
				}
			} else if !strings.Contains(warning, "certificate") || issuerRequests.Load() != 0 {
				t.Fatalf("untrusted issuer: requests=%d warning=%q", issuerRequests.Load(), warning)
			}
		})
	}
}

func TestBundledIssuerLoginRefreshAndLogout(t *testing.T) {
	var mutex sync.Mutex
	var metadata Metadata
	var grants []string
	var revocations int
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mutex.Lock()
			defer mutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case discoveryPath:
				_ = json.NewEncoder(w).Encode(metadata)
			case "/keycloak/device":
				_ = json.NewEncoder(w).Encode(DevicePrompt{
					DeviceCode: "device", UserCode: "ABCD", ExpiresIn: 60,
					VerificationURI: metadata.Issuer + "/verify", Interval: 1,
				})
			case "/keycloak/token":
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				grants = append(grants, r.Form.Get("grant_type"))
				_, _ = w.Write(
					[]byte(
						`{"access_token":"access","refresh_token":"refresh","expires_in":0}`,
					),
				)
			case "/keycloak/revoke":
				revocations++
				_, _ = w.Write([]byte(`{}`))
			default:
				t.Errorf("unexpected request to %s", r.URL.Path)
			}
		}),
	)
	defer server.Close()
	mutex.Lock()
	metadata = Metadata{
		Issuer:                      server.URL + "/keycloak",
		AuthorizationEndpoint:       server.URL + "/keycloak/authorize",
		TokenEndpoint:               server.URL + "/keycloak/token",
		DeviceAuthorizationEndpoint: server.URL + "/keycloak/device",
		RevocationEndpoint:          server.URL + "/keycloak/revoke",
		CLIClientID:                 "reana-client",
	}
	mutex.Unlock()
	t.Setenv(tlsVerifyEnv, "no")
	t.Setenv(caCertsEnv, "")
	t.Setenv(configPathEnv, t.TempDir()+"/credentials.json")
	t.Setenv(loginLoopbackPortEnv, "0")
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.HTTPClient.CloseIdleConnections()
	defer manager.IdPHTTPClient.CloseIdleConnections()
	manager.Sleep = func(context.Context, time.Duration) error { return nil }
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	credentials, err := manager.LoginBrowser(
		ctx,
		server.URL,
		func(string) {},
		func(raw string) error {
			parsed, err := url.Parse(raw)
			if err != nil {
				return err
			}
			query := parsed.Query()
			callback := query.Get("redirect_uri") + "?" + url.Values{
				"code": {"code"}, "state": {query.Get("state")},
			}.Encode()
			response, err := http.Get(callback)
			if response != nil {
				response.Body.Close()
			}
			return err
		},
	)
	if err != nil || credentials.AccessToken != "access" {
		t.Fatalf("browser login failed: %v", err)
	}
	credentials, err = manager.Refresh(ctx, server.URL, credentials)
	if err != nil || credentials.AccessToken != "access" {
		t.Fatalf("refresh failed: %v", err)
	}
	if warning, err := manager.Logout(ctx, server.URL); err != nil ||
		warning != "" {
		t.Fatalf("logout failed: %v %s", err, warning)
	}
	credentials, err = manager.LoginDevice(
		ctx,
		server.URL,
		func(DevicePrompt) {},
	)
	if err != nil || credentials.AccessToken != "access" {
		t.Fatalf("device login failed: %v", err)
	}
	if warning := manager.revokeBestEffort(ctx, server.URL, metadata, "discarded"); warning != "" {
		t.Fatal(warning)
	}
	want := "authorization_code,refresh_token,urn:ietf:params:oauth:grant-type:device_code"
	mutex.Lock()
	defer mutex.Unlock()
	if strings.Join(grants, ",") != want || revocations != 2 {
		t.Fatalf("grants=%v revocations=%d", grants, revocations)
	}
}

func TestSameHTTPSOrigin(t *testing.T) {
	cases := []struct {
		server, endpoint string
		same             bool
	}{
		{
			"https://localhost:30443",
			"https://localhost:30443/keycloak/token",
			true,
		},
		{"https://LOCALHOST", "https://localhost:443/keycloak", true},
		{"https://localhost:0443", "https://localhost/keycloak", true},
		{"https://[::1]:30443", "https://[::1]:30443/keycloak", true},
		{"https://localhost", "https://localhost:30443/keycloak", false},
		{"https://localhost", "https://127.0.0.1/keycloak", false},
		{"https://localhost", "https://localhost.example.org/keycloak", false},
		{"https://localhost", "http://localhost/keycloak", false},
		{"http://localhost", "https://localhost/keycloak", false},
		{"https://localhost", "https://localhost@evil.example/keycloak", false},
		{"https://localhost", "https://user@localhost/keycloak", false},
		{"https://localhost:0", "https://localhost:0/keycloak", false},
		{"https://localhost:65536", "https://localhost:65536/keycloak", false},
		{"https://localhost:bad", "https://localhost:bad/keycloak", false},
		{"", "https://localhost/keycloak", false},
	}
	for _, tc := range cases {
		t.Run(tc.server+" to "+tc.endpoint, func(t *testing.T) {
			if got := sameHTTPSOrigin(tc.server, tc.endpoint); got != tc.same {
				t.Fatalf("sameHTTPSOrigin=%v, want %v", got, tc.same)
			}
		})
	}
}
