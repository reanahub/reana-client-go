/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package client

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/reanahub/reana-client-go/client/operations"
	"github.com/reanahub/reana-client-go/pkg/auth"
)

func setServerURL(t *testing.T, serverURL string) {
	t.Helper()
	viper.Set("server-url", serverURL)
	t.Cleanup(viper.Reset)
}

func TestAPIClientUsesStoredOIDCToken(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer stored.jwt.token" {
				t.Errorf("Authorization = %q, want stored token", got)
			}
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()
	setServerURL(t, server.URL)
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	store, err := auth.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(server.URL, auth.Credentials{
		AccessToken:          "stored.jwt.token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	api, err := ApiClient("")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = api.Operations.GetYou(operations.NewGetYouParams(), nil)
}

func TestAPIClientUsesBearerWithoutQueryOverLoopbackHTTP(t *testing.T) {
	requestReceived := false
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true
			if got := r.Header.Get("Authorization"); got != "Bearer jwt-override" {
				t.Errorf(
					"Authorization = %q, want %q",
					got,
					"Bearer jwt-override",
				)
			}
			if r.URL.Query().Has("access_token") {
				t.Errorf("access_token leaked into query: %s", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()
	setServerURL(t, server.URL)

	api, err := ApiClient("jwt-override")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = api.Operations.GetYou(operations.NewGetYouParams(), nil)
	if !requestReceived {
		t.Fatal("HTTP server did not receive generated client request")
	}
}

func TestAPIClientRejectsBearerOverNonLoopbackHTTP(t *testing.T) {
	setServerURL(t, "http://reana.example")

	_, err := ApiClient("jwt")
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected cleartext bearer rejection, got %v", err)
	}
}

func TestAPIClientRejectsAllRedirects(t *testing.T) {
	setServerURL(t, "https://reana.example")
	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}

	redirect, _ := http.NewRequest(
		http.MethodGet,
		"http://reana.example/api/ping",
		nil,
	)
	original, _ := http.NewRequest(
		http.MethodGet,
		"https://reana.example/api/ping",
		nil,
	)
	err = api.httpClient.CheckRedirect(redirect, []*http.Request{original})
	if err != http.ErrUseLastResponse {
		t.Fatalf("redirect policy error = %v, want ErrUseLastResponse", err)
	}
}

func TestAPIClientDoesNotReplayBearerAcrossRedirect(t *testing.T) {
	targetRequests := 0
	target := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			targetRequests++
			if authorization := r.Header.Get("Authorization"); authorization != "" {
				t.Errorf(
					"redirect target received Authorization %q",
					authorization,
				)
			}
			w.WriteHeader(http.StatusOK)
		},
	))
	defer target.Close()
	origin := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(
				w,
				r,
				target.URL+"/captured",
				http.StatusTemporaryRedirect,
			)
		},
	))
	defer origin.Close()
	setServerURL(t, origin.URL)
	t.Setenv("REANA_SERVER_TLS_VERIFY", "false")

	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = api.Operations.GetYou(operations.NewGetYouParams(), nil)
	if targetRequests != 0 {
		t.Fatalf("redirect target received %d requests", targetRequests)
	}
}

func TestAPIClientVerifiesTLSCertificates(t *testing.T) {
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()
	setServerURL(t, server.URL)

	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.Operations.GetYou(operations.NewGetYouParams(), nil)
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("expected certificate verification error, got %v", err)
	}
}

func TestAPIClientSupportsExplicitInsecureTLSForLocalTesting(t *testing.T) {
	requestReceived := false
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestReceived = true
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()
	setServerURL(t, server.URL)
	t.Setenv("REANA_SERVER_TLS_VERIFY", "false")

	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = api.Operations.GetYou(operations.NewGetYouParams(), nil)
	if !requestReceived {
		t.Fatal("explicit insecure TLS setting did not reach the test server")
	}
}

func TestAPIClientSupportsCustomCABundle(t *testing.T) {
	requestReceived := false
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestReceived = true
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()
	setServerURL(t, server.URL)
	certificate := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	caPath := t.TempDir() + "/ca.pem"
	if err := os.WriteFile(caPath, certificate, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REANA_SERVER_CA_CERTS", caPath)

	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = api.Operations.GetYou(operations.NewGetYouParams(), nil)
	if !requestReceived {
		t.Fatal("custom CA bundle did not reach the test server")
	}
}

func TestAPIClientDisablesGoOpenAPIDumpsAtDebugLevel(t *testing.T) {
	oldLevel := log.GetLevel()
	log.SetLevel(log.DebugLevel)
	t.Cleanup(func() { log.SetLevel(oldLevel) })
	setServerURL(t, "https://reana.example")

	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := api.Transport.(*httptransport.Runtime)
	if !ok {
		t.Fatalf("transport type = %T, want *client.Runtime", api.Transport)
	}
	if transport.Debug {
		t.Fatal("go-openapi request dumping is enabled")
	}
}

func TestStreamingHTTPClientReturnsBoundedClientWithNoTimeout(t *testing.T) {
	setServerURL(t, "https://reana.example")

	httpClient, u, err := StreamingHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	if httpClient.Timeout != 0 {
		t.Fatalf(
			"Timeout = %v, want 0 (no whole-request deadline)",
			httpClient.Timeout,
		)
	}
	if u.Host != "reana.example" {
		t.Fatalf("host = %q, want reana.example", u.Host)
	}
	transport, ok := httpClient.Transport.(*boundedResponseTransport)
	if !ok {
		t.Fatalf(
			"transport type = %T, want *boundedResponseTransport",
			httpClient.Transport,
		)
	}
	inner, ok := transport.transport.(*http.Transport)
	if !ok {
		t.Fatalf(
			"inner transport type = %T, want *http.Transport",
			transport.transport,
		)
	}
	if inner.ResponseHeaderTimeout != apiRequestTimeout {
		t.Fatalf(
			"ResponseHeaderTimeout = %v, want %v",
			inner.ResponseHeaderTimeout,
			apiRequestTimeout,
		)
	}
}

func TestStreamingHTTPClientRejectsMissingServerURL(t *testing.T) {
	setServerURL(t, "")

	_, _, err := StreamingHTTPClient()
	if err == nil ||
		!strings.Contains(err.Error(), "REANA server URL is not set") {
		t.Fatalf("expected missing server URL error, got %v", err)
	}
}

func TestStreamingHTTPClientRejectsCleartextNonLoopback(t *testing.T) {
	setServerURL(t, "http://reana.example")

	_, _, err := StreamingHTTPClient()
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("expected cleartext bearer rejection, got %v", err)
	}
}

func TestGetInteractiveSessionSecret(t *testing.T) {
	workflow := "analysis/run 1?"
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wantPath := "/api/workflows/analysis%2Frun%201%3F/interactive-session-secret"
			if got := r.URL.EscapedPath(); got != wantPath {
				t.Errorf("escaped path = %q, want %q", got, wantPath)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer jwt" {
				t.Errorf("Authorization = %q, want %q", got, "Bearer jwt")
			}
			if r.URL.RawQuery != "" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(
				[]byte(`{"session_secret":"short-lived+/=","path":"/session"}`),
			)
		}),
	)
	defer server.Close()
	setServerURL(t, server.URL)

	api, err := ApiClient("jwt")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := api.GetInteractiveSessionSecret(
		context.Background(),
		workflow,
	)
	if err != nil {
		t.Fatal(err)
	}
	if secret != "short-lived+/=" {
		t.Fatalf("session secret = %q, want %q", secret, "short-lived+/=")
	}
}
