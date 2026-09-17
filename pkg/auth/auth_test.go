/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return function(request)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testManager(t *testing.T, handler roundTripFunc) *Manager {
	t.Helper()
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	httpClient := &http.Client{
		Transport: handler,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Manager{
		Store: &Store{
			Path: filepath.Join(t.TempDir(), "reana-client.json"),
		},
		HTTPClient:    httpClient,
		IdPHTTPClient: httpClient,
		Now:           func() time.Time { return now },
		Sleep:         func(context.Context, time.Duration) error { return nil },
	}
}

func managerWithStore(path string, handler roundTripFunc) *Manager {
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	httpClient := &http.Client{
		Transport: handler,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Manager{
		Store:         &Store{Path: path},
		HTTPClient:    httpClient,
		IdPHTTPClient: httpClient,
		Now:           func() time.Time { return now },
		Sleep:         func(context.Context, time.Duration) error { return nil },
	}
}

func expiredCredentials(now time.Time) Credentials {
	return Credentials{
		Issuer:               "https://issuer.example.org",
		ClientID:             "reana-cli",
		TokenEndpoint:        "https://issuer.example.org/token",
		AccessToken:          "old.access.token",
		AccessTokenExpiresAt: now.Add(-time.Hour).Format(time.RFC3339),
		RefreshToken:         "old-refresh",
	}
}

func TestStoreUsesPythonCompatibleSchemaAndPermissions(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "reana-client.json")}
	stored, err := store.Put("HTTPS://reana.example.org/", Credentials{
		Issuer:      "https://issuer.example.org",
		ClientID:    "reana-cli",
		AccessToken: "access",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if stored.CredentialEpoch != 1 {
		t.Fatalf("credential epoch = %d, want 1", stored.CredentialEpoch)
	}
	contents, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(contents, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["active_server"] != "https://reana.example.org" {
		t.Fatalf("active server = %v", raw["active_server"])
	}
	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("credential permissions = %o, want 600", permissions)
	}
}

func TestDefaultCredentialDirectoryHasRestrictivePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("REANA_CLIENT_CONFIG", "")
	directory := filepath.Join(home, ".config", "reana")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	if store.Path != filepath.Join(directory, "reana-client.json") {
		t.Fatalf("default credential path = %q", store.Path)
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("credential directory permissions = %o, want 700", permissions)
	}
}

func TestNormalizeServerURL(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{"reana.example.org/", "https://reana.example.org"},
		{"https://reana.example.org/base/", "https://reana.example.org/base"},
		{"http://127.0.0.1:5000/", "http://127.0.0.1:5000"},
		// DNS hostnames are case-insensitive; two URLs differing only in host
		// casing must normalize to the same credential-store key.
		{"https://REANA.example.org/", "https://reana.example.org"},
	} {
		got, err := NormalizeServerURL(test.input)
		if err != nil || got != test.want {
			t.Errorf(
				"NormalizeServerURL(%q) = %q, %v; want %q",
				test.input,
				got,
				err,
				test.want,
			)
		}
	}
	if _, err := NormalizeServerURL("http://reana.example.org"); err == nil {
		t.Fatal("non-loopback HTTP server was accepted")
	}
}

func TestDiscoverValidatesRelayedMetadata(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != discoveryPath {
				t.Fatalf("discovery path = %q", request.URL.Path)
			}
			return jsonResponse(http.StatusOK, `{
            "issuer":"https://issuer.example.org",
            "authorization_endpoint":"https://issuer.example.org/auth",
            "token_endpoint":"https://issuer.example.org/token",
            "device_authorization_endpoint":"https://issuer.example.org/device",
            "revocation_endpoint":"https://issuer.example.org/revoke",
            "reana_cli_client_id":"reana-cli"
        }`), nil
		},
	)
	metadata, err := manager.Discover(
		context.Background(),
		"https://reana.example.org",
	)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.CLIClientID != "reana-cli" {
		t.Fatalf("client id = %q", metadata.CLIClientID)
	}
}

func TestDiscoverRejectsCredentialEndpointRedirect(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		response := jsonResponse(http.StatusTemporaryRedirect, `{}`)
		response.Header.Set("Location", "http://attacker.example/metadata")
		return response, nil
	})
	_, err := manager.Discover(
		context.Background(),
		"https://reana.example.org",
	)
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
}

func TestCredentialPostsAreNotReplayedAcrossRedirects(t *testing.T) {
	targetRequests := atomic.Int32{}
	target := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, request *http.Request) {
			targetRequests.Add(1)
			_, _ = io.ReadAll(request.Body)
			w.WriteHeader(http.StatusOK)
		},
	))
	defer target.Close()
	origin := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, request *http.Request) {
			http.Redirect(w, request, target.URL, http.StatusTemporaryRedirect)
		},
	))
	defer origin.Close()
	httpClient := origin.Client()
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	manager := &Manager{
		Store: &Store{
			Path: filepath.Join(t.TempDir(), "credentials.json"),
		},
		HTTPClient:    httpClient,
		IdPHTTPClient: httpClient,
		Now:           time.Now,
		Sleep:         func(context.Context, time.Duration) error { return nil },
	}

	var tokens tokenResponse
	response, err := manager.postForm(
		context.Background(),
		"https://reana.example.org",
		origin.URL,
		"token refresh",
		url.Values{"refresh_token": {"secret-refresh"}},
		&tokens,
	)
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf(
			"expected token redirect rejection, got response=%v err=%v",
			response,
			err,
		)
	}
	warning := manager.revokeBestEffort(
		context.Background(),
		"https://reana.example.org",
		Metadata{
			CLIClientID:        "reana-cli",
			RevocationEndpoint: origin.URL,
		},
		"secret-refresh",
	)
	if !strings.Contains(warning, "redirect") {
		t.Fatalf("revocation warning = %q", warning)
	}
	if targetRequests.Load() != 0 {
		t.Fatalf(
			"redirect target received %d credential posts",
			targetRequests.Load(),
		)
	}
}

func TestDeviceLoginUsesPKCEAndStoresTokens(t *testing.T) {
	requests := 0
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			requests++
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "device_authorization_endpoint":"https://issuer.example.org/device",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/device":
				form := readForm(t, request)
				if form.Get("code_challenge") == "" ||
					form.Get("code_challenge_method") != "S256" {
					t.Fatalf("device PKCE form = %v", form)
				}
				return jsonResponse(
					http.StatusOK,
					`{"device_code":"device","user_code":"ABCD","verification_uri":"https://issuer.example.org/verify","expires_in":300,"interval":1}`,
				), nil
			case "/token":
				form := readForm(t, request)
				if form.Get("code_verifier") == "" ||
					form.Get(
						"grant_type",
					) != "urn:ietf:params:oauth:grant-type:device_code" {
					t.Fatalf("device token form = %v", form)
				}
				return jsonResponse(
					http.StatusOK,
					`{"access_token":"a.b.c","refresh_token":"refresh","expires_in":3600}`,
				), nil
			default:
				t.Fatalf("unexpected request: %s", request.URL)
				return nil, nil
			}
		},
	)
	prompted := false
	credentials, err := manager.LoginDevice(
		context.Background(),
		"https://reana.example.org",
		func(prompt DevicePrompt) {
			prompted = prompt.UserCode == "ABCD"
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !prompted || credentials.AccessToken != "a.b.c" ||
		credentials.RefreshToken != "refresh" ||
		requests != 3 {
		t.Fatalf(
			"unexpected device result: prompted=%v credentials=%+v requests=%d",
			prompted,
			credentials,
			requests,
		)
	}
}

func TestBrowserLoginValidatesStateAndUsesPKCE(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/token":
				form := readForm(t, request)
				if form.Get("grant_type") != "authorization_code" ||
					form.Get("code_verifier") == "" ||
					form.Get("code") != "code" {
					t.Fatalf("authorization code form = %v", form)
				}
				return jsonResponse(
					http.StatusOK,
					`{"access_token":"browser.jwt.token","refresh_token":"refresh","expires_in":3600}`,
				), nil
			default:
				t.Fatalf("unexpected request: %s", request.URL)
				return nil, nil
			}
		},
	)
	var displayed string
	credentials, err := manager.LoginBrowser(
		context.Background(),
		"https://reana.example.org",
		func(value string) { displayed = value },
		func(authorizationURL string) error {
			parsed, err := url.Parse(authorizationURL)
			if err != nil {
				return err
			}
			if parsed.Query().Get("code_challenge_method") != "S256" ||
				parsed.Query().Get("code_challenge") == "" {
				t.Fatalf("authorization URL lacks PKCE: %s", authorizationURL)
			}
			callback := parsed.Query().
				Get("redirect_uri") +
				"?code=code&state=" + url.QueryEscape(
				parsed.Query().Get("state"),
			)
			response, err := http.Get(callback) //nolint:gosec
			if err == nil {
				response.Body.Close()
			}
			return err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if displayed == "" || credentials.AccessToken != "browser.jwt.token" {
		t.Fatalf(
			"unexpected browser login result: displayed=%q credentials=%+v",
			displayed,
			credentials,
		)
	}
}

func TestBrowserLoginUsesConfiguredLoopbackPort(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()
	t.Setenv(loginLoopbackPortEnv, strconv.Itoa(port))

	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/token":
				return jsonResponse(
					http.StatusOK,
					`{"access_token":"browser.jwt.token","refresh_token":"refresh","expires_in":3600}`,
				), nil
			default:
				return nil, fmt.Errorf("unexpected request: %s", request.URL)
			}
		},
	)
	_, err = manager.LoginBrowser(
		context.Background(),
		"https://reana.example.org",
		func(string) {},
		func(authorizationURL string) error {
			parsed, parseErr := url.Parse(authorizationURL)
			if parseErr != nil {
				return parseErr
			}
			redirect, parseErr := url.Parse(parsed.Query().Get("redirect_uri"))
			if parseErr != nil {
				return parseErr
			}
			if redirect.Port() != strconv.Itoa(port) {
				return fmt.Errorf(
					"redirect port = %q, want %d",
					redirect.Port(),
					port,
				)
			}
			callback := redirect.String() + "?code=code&state=" +
				url.QueryEscape(parsed.Query().Get("state"))
			response, requestErr := http.Get(callback) //nolint:gosec
			if requestErr == nil {
				response.Body.Close()
			}
			return requestErr
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrowserLoginRejectsInvalidConfiguredLoopbackPort(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
            "issuer":"https://issuer.example.org",
            "authorization_endpoint":"https://issuer.example.org/auth",
            "token_endpoint":"https://issuer.example.org/token",
            "reana_cli_client_id":"reana-cli"
        }`), nil
		},
	)
	for _, value := range []string{"not-a-port", "70000", "-1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(loginLoopbackPortEnv, value)
			_, err := manager.LoginBrowser(
				context.Background(),
				"https://reana.example.org",
				func(string) {},
				func(string) error { return nil },
			)
			if err == nil ||
				!strings.Contains(err.Error(), loginLoopbackPortEnv) {
				t.Fatalf("invalid port error = %v", err)
			}
		})
	}
}

func TestBrowserLoginReportsOccupiedConfiguredLoopbackPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	t.Setenv(loginLoopbackPortEnv, strconv.Itoa(port))
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
            "issuer":"https://issuer.example.org",
            "authorization_endpoint":"https://issuer.example.org/auth",
            "token_endpoint":"https://issuer.example.org/token",
            "reana_cli_client_id":"reana-cli"
        }`), nil
		},
	)

	_, err = manager.LoginBrowser(
		context.Background(),
		"https://reana.example.org",
		func(string) {},
		func(string) error { return nil },
	)
	if err == nil || !strings.Contains(err.Error(), loginLoopbackPortEnv) {
		t.Fatalf("occupied port error = %v", err)
	}
}

func TestAccessTokenRefreshesAndPreservesRotatedRefreshToken(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			form := readForm(t, request)
			if form.Get("grant_type") != "refresh_token" ||
				form.Get("refresh_token") != "old-refresh" {
				t.Fatalf("refresh form = %v", form)
			}
			return jsonResponse(
				http.StatusOK,
				`{"access_token":"new.access.token","expires_in":3600}`,
			), nil
		},
	)
	_, err := manager.Store.Put("https://reana.example.org", Credentials{
		Issuer:                "https://issuer.example.org",
		ClientID:              "reana-cli",
		TokenEndpoint:         "https://issuer.example.org/token",
		AuthorizationEndpoint: "https://issuer.example.org/auth",
		AccessToken:           "old.access.token",
		AccessTokenExpiresAt: manager.Now().
			Add(10 * time.Second).
			Format(time.RFC3339),
		RefreshToken: "old-refresh",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.AccessToken(
		context.Background(),
		"https://reana.example.org",
	)
	if err != nil {
		t.Fatal(err)
	}
	if token != "new.access.token" {
		t.Fatalf("access token = %q", token)
	}
	stored, _ := manager.Store.Get("https://reana.example.org")
	if stored.RefreshToken != "old-refresh" {
		t.Fatalf(
			"refresh token = %q, want preserved token",
			stored.RefreshToken,
		)
	}
}

func TestConcurrentManagersPerformOneRefresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reana-client.json")
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	handler := roundTripFunc(
		func(request *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				close(started)
				<-release
			}
			return jsonResponse(
				http.StatusOK,
				`{"access_token":"new.access.token","refresh_token":"new-refresh","expires_in":3600}`,
			), nil
		},
	)
	first := managerWithStore(path, handler)
	second := managerWithStore(path, handler)
	if _, err := first.Store.Put(
		"https://reana.example.org", expiredCredentials(first.Now()), true,
	); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	go func() {
		_, err := first.AccessToken(
			context.Background(),
			"https://reana.example.org",
		)
		results <- err
	}()
	<-started
	go func() {
		_, err := second.AccessToken(
			context.Background(),
			"https://reana.example.org",
		)
		results <- err
	}()
	close(release)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("refresh calls = %d, want 1", calls.Load())
	}
}

func TestLoginDuringRefreshPreventsStaleWriteBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reana-client.json")
	started := make(chan struct{})
	release := make(chan struct{})
	handler := roundTripFunc(
		func(request *http.Request) (*http.Response, error) {
			close(started)
			<-release
			return jsonResponse(
				http.StatusOK,
				`{"access_token":"stale.access.token","refresh_token":"stale-refresh","expires_in":3600}`,
			), nil
		},
	)
	manager := managerWithStore(path, handler)
	if _, err := manager.Store.Put(
		"https://reana.example.org", expiredCredentials(manager.Now()), true,
	); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := manager.Refresh(
			context.Background(), "https://reana.example.org", Credentials{},
		)
		result <- err
	}()
	<-started
	winner := expiredCredentials(manager.Now())
	winner.AccessToken = "login.access.token"
	winner.RefreshToken = "login-refresh"
	if _, err := manager.Store.Put("https://reana.example.org", winner, true); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-result; err == nil {
		t.Fatal("stale refresh unexpectedly succeeded")
	}
	stored, err := manager.Store.Get("https://reana.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if stored.RefreshToken != "login-refresh" {
		t.Fatalf("refresh token = %q, want login-refresh", stored.RefreshToken)
	}
}

func TestLogoutDuringRefreshPreventsCredentialResurrection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reana-client.json")
	started := make(chan struct{})
	release := make(chan struct{})
	handler := roundTripFunc(
		func(request *http.Request) (*http.Response, error) {
			close(started)
			<-release
			return jsonResponse(
				http.StatusOK,
				`{"access_token":"stale.access.token","refresh_token":"stale-refresh","expires_in":3600}`,
			), nil
		},
	)
	manager := managerWithStore(path, handler)
	if _, err := manager.Store.Put(
		"https://reana.example.org", expiredCredentials(manager.Now()), true,
	); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := manager.Refresh(
			context.Background(), "https://reana.example.org", Credentials{},
		)
		result <- err
	}()
	<-started
	if _, err := manager.Store.Logout(
		"https://reana.example.org", func(Credentials) string { return "" },
	); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-result; err == nil {
		t.Fatal("stale refresh unexpectedly succeeded")
	}
	stored, err := manager.Store.Get("https://reana.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if stored.AccessToken != "" || stored.RefreshToken != "" {
		t.Fatalf("logout credentials were resurrected: %+v", stored)
	}
}

func TestInvalidGrantDoesNotClearNewerCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reana-client.json")
	started := make(chan struct{})
	release := make(chan struct{})
	handler := roundTripFunc(
		func(request *http.Request) (*http.Response, error) {
			close(started)
			<-release
			return jsonResponse(
				http.StatusBadRequest,
				`{"error":"invalid_grant"}`,
			), nil
		},
	)
	manager := managerWithStore(path, handler)
	if _, err := manager.Store.Put(
		"https://reana.example.org", expiredCredentials(manager.Now()), true,
	); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := manager.Refresh(
			context.Background(), "https://reana.example.org", Credentials{},
		)
		result <- err
	}()
	<-started
	winner := expiredCredentials(manager.Now())
	winner.AccessToken = "winner.access.token"
	winner.RefreshToken = "winner-refresh"
	if _, err := manager.Store.Put("https://reana.example.org", winner, true); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-result; err == nil ||
		!strings.Contains(err.Error(), "changed") {
		t.Fatalf("invalid_grant race error = %v", err)
	}
	stored, err := manager.Store.Get("https://reana.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if stored.RefreshToken != "winner-refresh" {
		t.Fatalf("newer credentials were cleared: %+v", stored)
	}
}

func TestRefreshLockWaitTimesOutPredictably(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "reana-client.json")}
	lock, err := store.tryRefreshLock("https://reana.example.org")
	if err != nil {
		t.Fatal(err)
	}
	defer releaseLock(lock)
	started := time.Now()
	finished, err := store.waitRefreshLock(
		"https://reana.example.org", 25*time.Millisecond,
	)
	if err != nil {
		t.Fatal(err)
	}
	if finished {
		t.Fatal("refresh lock wait unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("refresh lock timeout took %s", elapsed)
	}
}

func TestLogoutAcceptsEmptyRevocationResponseAndClearsTokens(t *testing.T) {
	var manager *Manager
	manager = testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/revoke" {
				t.Fatalf("revocation path = %q", request.URL.Path)
			}
			probe, err := os.OpenFile(
				manager.Store.Path+".lock",
				os.O_CREATE|os.O_RDWR,
				0o600,
			)
			if err != nil {
				t.Fatal(err)
			}
			defer probe.Close()
			if err := unix.Flock(
				int(probe.Fd()),
				unix.LOCK_EX|unix.LOCK_NB,
			); err == nil {
				_ = unix.Flock(int(probe.Fd()), unix.LOCK_UN)
				t.Fatal("credential lock was not held during token revocation")
			}
			return jsonResponse(http.StatusNoContent, ""), nil
		},
	)
	_, err := manager.Store.Put("https://reana.example.org", Credentials{
		ClientID:           "reana-cli",
		RevocationEndpoint: "https://issuer.example.org/revoke",
		AccessToken:        "access",
		RefreshToken:       "refresh",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	warning, err := manager.Logout(
		context.Background(),
		"https://reana.example.org",
	)
	if err != nil || warning != "" {
		t.Fatalf("logout = warning %q, error %v", warning, err)
	}
	stored, _ := manager.Store.Get("https://reana.example.org")
	if stored.AccessToken != "" || stored.RefreshToken != "" {
		t.Fatalf("tokens were not cleared: %+v", stored)
	}
}

func TestNewManagerWiresStoreHTTPClientAndSleep(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("REANA_CLIENT_CONFIG", "")

	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if manager.Store == nil || manager.HTTPClient != nil ||
		manager.IdPHTTPClient != nil ||
		manager.Now == nil ||
		manager.Sleep == nil {
		t.Fatalf("manager was not fully wired: %+v", manager)
	}
	if manager.Now().IsZero() {
		t.Fatal("Now() returned the zero time")
	}
	if err := manager.Sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("Sleep did not complete normally: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.Sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Sleep on a cancelled context = %v, want context.Canceled",
			err,
		)
	}
}

func TestNewManagerPropagatesStoreError(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("REANA_CLIENT_CONFIG", "")
	if _, err := NewManager(); err == nil {
		t.Fatal(
			"expected an error when the credential store path cannot be resolved",
		)
	}
}

func TestTokenExpiry(t *testing.T) {
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	value := func(seconds int64) *int64 { return &seconds }
	encodeClaims := func(claims string) string {
		return "header." + base64.RawURLEncoding.EncodeToString(
			[]byte(claims),
		) + ".sig"
	}
	cases := []struct {
		name        string
		accessToken string
		expiresIn   *int64
		want        string
		wantError   bool
	}{
		{
			name:        "uses expires_in when positive",
			accessToken: "opaque-token",
			expiresIn:   value(3600),
			want:        now.Add(time.Hour).Format(time.RFC3339),
		},
		{
			name:        "honors explicit zero",
			accessToken: "opaque-token",
			expiresIn:   value(0),
			want:        now.Format(time.RFC3339),
		},
		{
			name:        "falls back to JWT exp claim",
			accessToken: encodeClaims(`{"exp":1755780000}`),
			want:        time.Unix(1755780000, 0).UTC().Format(time.RFC3339),
		},
		{
			name:        "not a compact JWT",
			accessToken: "opaque-token",
			want:        "",
		},
		{
			name:        "malformed base64 payload",
			accessToken: "header.not-valid-base64!!.sig",
			want:        "",
		},
		{
			name: "payload is not JSON",
			accessToken: "header." + base64.RawURLEncoding.EncodeToString(
				[]byte("not json"),
			) + ".sig",
			want: "",
		},
		{
			name:        "exp claim is not an integer",
			accessToken: encodeClaims(`{"exp":"soon"}`),
			want:        "",
		},
		{
			name:        "exp claim is missing",
			accessToken: encodeClaims(`{}`),
			want:        "",
		},
		{
			name:        "rejects negative lifetime",
			accessToken: "opaque-token",
			expiresIn:   value(-1),
			wantError:   true,
		},
		{
			name:        "rejects duration overflow",
			accessToken: "opaque-token",
			expiresIn: value(
				int64(math.MaxInt64/time.Second) + 1,
			),
			wantError: true,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := tokenExpiry(
				testCase.accessToken,
				testCase.expiresIn,
				now,
			)
			if (err != nil) != testCase.wantError {
				t.Fatalf(
					"tokenExpiry error = %v, wantError = %t",
					err,
					testCase.wantError,
				)
			}
			if got != testCase.want {
				t.Fatalf(
					"tokenExpiry(%q, %v) = %q, want %q",
					testCase.accessToken,
					testCase.expiresIn,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestAccessTokenValid(t *testing.T) {
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name        string
		credentials Credentials
		want        bool
	}{
		{"no access token", Credentials{}, false},
		{
			"no expiry recorded fails closed and triggers a refresh",
			Credentials{AccessToken: "a.b.c"},
			false,
		},
		{
			"expired",
			Credentials{
				AccessToken: "a.b.c",
				AccessTokenExpiresAt: now.Add(-time.Minute).
					Format(time.RFC3339),
			},
			false,
		},
		{
			"within expiry leeway",
			Credentials{
				AccessToken: "a.b.c",
				AccessTokenExpiresAt: now.Add(30 * time.Second).
					Format(time.RFC3339),
			},
			false,
		},
		{
			"valid",
			Credentials{
				AccessToken:          "a.b.c",
				AccessTokenExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
			},
			true,
		},
		{
			"unparsable expiry",
			Credentials{
				AccessToken:          "a.b.c",
				AccessTokenExpiresAt: "not-a-time",
			},
			false,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := accessTokenValid(testCase.credentials, now); got != testCase.want {
				t.Fatalf(
					"accessTokenValid(%+v) = %v, want %v",
					testCase.credentials,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestValidateMetadataRejectsMissingOrInvalidFields(t *testing.T) {
	valid := Metadata{
		Issuer:                "https://issuer.example.org",
		AuthorizationEndpoint: "https://issuer.example.org/auth",
		TokenEndpoint:         "https://issuer.example.org/token",
		CLIClientID:           "reana-cli",
	}
	if err := validateMetadata(valid, false); err != nil {
		t.Fatalf("expected valid metadata to pass, got %v", err)
	}

	cases := []struct {
		name           string
		mutate         func(Metadata) Metadata
		deviceRequired bool
	}{
		{
			"missing issuer",
			func(m Metadata) Metadata { m.Issuer = ""; return m },
			false,
		},
		{
			"missing authorization endpoint",
			func(m Metadata) Metadata { m.AuthorizationEndpoint = ""; return m },
			false,
		},
		{
			"missing token endpoint",
			func(m Metadata) Metadata { m.TokenEndpoint = ""; return m },
			false,
		},
		{
			"missing client id",
			func(m Metadata) Metadata { m.CLIClientID = ""; return m },
			false,
		},
		{
			"missing device endpoint when required",
			func(m Metadata) Metadata { return m },
			true,
		},
		{
			"invalid revocation endpoint",
			func(m Metadata) Metadata { m.RevocationEndpoint = "http://insecure"; return m },
			false,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := testCase.mutate(valid)
			if err := validateMetadata(metadata, testCase.deviceRequired); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestValidateStoredMetadataRejectsMissingOrInvalidFields(t *testing.T) {
	valid := Metadata{
		Issuer:        "https://issuer.example.org",
		TokenEndpoint: "https://issuer.example.org/token",
		CLIClientID:   "reana-cli",
	}
	if err := validateStoredMetadata(valid); err != nil {
		t.Fatalf("expected valid stored metadata to pass, got %v", err)
	}

	cases := []struct {
		name   string
		mutate func(Metadata) Metadata
	}{
		{
			"missing issuer",
			func(m Metadata) Metadata { m.Issuer = ""; return m },
		},
		{
			"missing token endpoint",
			func(m Metadata) Metadata { m.TokenEndpoint = ""; return m },
		},
		{
			"missing client id",
			func(m Metadata) Metadata { m.CLIClientID = ""; return m },
		},
		{
			"invalid authorization endpoint",
			func(m Metadata) Metadata {
				m.AuthorizationEndpoint = "not-a-url"
				return m
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := testCase.mutate(valid)
			if err := validateStoredMetadata(metadata); err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !IsAuthenticationError(validateStoredMetadata(metadata)) {
				t.Fatal("expected an AuthenticationError")
			}
		})
	}
}

func TestCredentialsFromTokenRequiresAccessToken(t *testing.T) {
	_, err := credentialsFromToken(
		Metadata{},
		tokenResponse{},
		"old-refresh",
		time.Now(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "did not return an access token") {
		t.Fatalf("expected missing access token error, got %v", err)
	}
}

func TestCredentialsFromTokenHonorsExplicitZeroLifetimes(t *testing.T) {
	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	var tokens tokenResponse
	if err := json.Unmarshal([]byte(`{
        "access_token":"opaque-token",
        "refresh_token":"refresh-token",
        "expires_in":0,
        "refresh_expires_in":0
    }`), &tokens); err != nil {
		t.Fatal(err)
	}
	credentials, err := credentialsFromToken(Metadata{}, tokens, "", now)
	if err != nil {
		t.Fatal(err)
	}
	want := now.Format(time.RFC3339)
	if credentials.AccessTokenExpiresAt != want {
		t.Fatalf(
			"access expiry = %q, want %q",
			credentials.AccessTokenExpiresAt,
			want,
		)
	}
	if credentials.RefreshTokenExpiresAt != want {
		t.Fatalf(
			"refresh expiry = %q, want %q",
			credentials.RefreshTokenExpiresAt,
			want,
		)
	}
	if accessTokenValid(credentials, now) {
		t.Fatal("zero-lifetime access token unexpectedly considered valid")
	}
}

func TestCredentialsFromTokenRejectsInvalidRefreshLifetime(t *testing.T) {
	invalid := int64(-1)
	_, err := credentialsFromToken(
		Metadata{},
		tokenResponse{
			AccessToken:      "opaque-token",
			RefreshExpiresIn: &invalid,
		},
		"",
		time.Now(),
	)
	if err == nil || !strings.Contains(err.Error(), "refresh_expires_in") {
		t.Fatalf("invalid refresh lifetime error = %v", err)
	}
}

func TestStoreTokensPreservesRotatedRefreshTokenWhenResponseIsRejected(
	t *testing.T,
) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("unexpected request")
	})
	invalidLifetime := int64(-1)
	_, err := manager.storeTokens(
		"https://reana.example.org",
		Metadata{
			Issuer:        "https://issuer.example.org",
			CLIClientID:   "reana-cli",
			TokenEndpoint: "https://issuer.example.org/token",
		},
		tokenResponse{
			AccessToken:  "rejected-access",
			RefreshToken: "rotated-refresh",
			ExpiresIn:    &invalidLifetime,
		},
		"old-refresh",
		false,
	)
	if err == nil || !strings.Contains(err.Error(), "expires_in") {
		t.Fatalf("invalid token response error = %v", err)
	}
	stored, getErr := manager.Store.Get("https://reana.example.org")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.RefreshToken != "rotated-refresh" {
		t.Fatalf(
			"refresh token = %q, want rotated-refresh",
			stored.RefreshToken,
		)
	}
	if stored.AccessToken != "" {
		t.Fatalf("rejected access token was stored: %q", stored.AccessToken)
	}
}

func TestRefreshPreservesRotatedRefreshTokenWhenResponseIsRejected(
	t *testing.T,
) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
            "access_token":"rejected-access",
            "refresh_token":"rotated-refresh",
            "expires_in":-1
        }`), nil
	})
	serverURL := "https://reana.example.org"
	credentials, err := manager.Store.Put(
		serverURL,
		expiredCredentials(manager.Now()),
		true,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.Refresh(context.Background(), serverURL, credentials)
	if err == nil || !strings.Contains(err.Error(), "expires_in") {
		t.Fatalf("invalid token response error = %v", err)
	}
	stored, getErr := manager.Store.Get(serverURL)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.RefreshToken != "rotated-refresh" {
		t.Fatalf(
			"refresh token = %q, want rotated-refresh",
			stored.RefreshToken,
		)
	}
	if stored.AccessToken != "" {
		t.Fatalf("rejected access token was stored: %q", stored.AccessToken)
	}
}

func TestAccessTokenReturnsCachedTokenWithoutRefreshing(t *testing.T) {
	called := false
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusOK, `{}`), nil
	})
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		AccessToken: "cached.access.token",
		AccessTokenExpiresAt: manager.Now().
			Add(time.Hour).
			Format(time.RFC3339),
	}, true); err != nil {
		t.Fatal(err)
	}
	token, err := manager.AccessToken(
		context.Background(),
		"https://reana.example.org",
	)
	if err != nil {
		t.Fatal(err)
	}
	if token != "cached.access.token" {
		t.Fatalf("token = %q, want cached token", token)
	}
	if called {
		t.Fatal("expected no network call for a still-valid cached token")
	}
}

func TestAccessTokenUsesActiveServerWhenServerURLEmpty(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected network call")
		return nil, nil
	})
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		AccessToken: "cached.access.token",
		AccessTokenExpiresAt: manager.Now().
			Add(time.Hour).
			Format(time.RFC3339),
	}, true); err != nil {
		t.Fatal(err)
	}
	token, err := manager.AccessToken(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if token != "cached.access.token" {
		t.Fatalf("token = %q, want cached token", token)
	}
}

func TestAccessTokenErrorsWhenNoActiveServer(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected network call")
		return nil, nil
	})
	_, err := manager.AccessToken(context.Background(), "")
	if err == nil ||
		!strings.Contains(err.Error(), "No REANA server is configured") {
		t.Fatalf("expected no-active-cluster error, got %v", err)
	}
}

func TestLogoutUsesActiveServerWhenServerURLEmpty(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusNoContent, ""), nil
		},
	)
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		RevocationEndpoint: "https://issuer.example.org/revoke",
		AccessToken:        "access",
		RefreshToken:       "refresh",
	}, true); err != nil {
		t.Fatal(err)
	}
	warning, err := manager.Logout(context.Background(), "")
	if err != nil || warning != "" {
		t.Fatalf("logout = warning %q, error %v", warning, err)
	}
	stored, _ := manager.Store.Get("https://reana.example.org")
	if stored.AccessToken != "" {
		t.Fatalf("tokens were not cleared: %+v", stored)
	}
}

func TestLogoutErrorsWhenNoActiveServer(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected network call")
		return nil, nil
	})
	_, err := manager.Logout(context.Background(), "")
	if err == nil ||
		!strings.Contains(err.Error(), "No REANA server is configured") {
		t.Fatalf("expected no-active-cluster error, got %v", err)
	}
}

func TestDiscoverReturnsConnectionErrorWhenServerUnreachable(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	_, err := manager.Discover(
		context.Background(),
		"https://reana.example.org",
	)
	if err == nil || !strings.Contains(err.Error(), "Could not connect") {
		t.Fatalf("expected connection error, got %v", err)
	}
}

func TestDiscoverReturnsErrorForNonSuccessStatus(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusInternalServerError, `{}`), nil
	})
	_, err := manager.Discover(
		context.Background(),
		"https://reana.example.org",
	)
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected HTTP 500 discovery error, got %v", err)
	}
}

func TestDiscoverRejectsInvalidMetadata(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(
			http.StatusOK,
			`{"authorization_endpoint":"https://issuer.example.org/auth","token_endpoint":"https://issuer.example.org/token","reana_cli_client_id":"reana-cli"}`,
		), nil
	})
	_, err := manager.Discover(
		context.Background(),
		"https://reana.example.org",
	)
	if !IsAuthenticationError(err) {
		t.Fatalf(
			"expected an AuthenticationError for missing issuer, got %v",
			err,
		)
	}
}

func TestRevokeBestEffortSkipsWhenEndpointOrTokenMissing(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected network call")
		return nil, nil
	})
	if warning := manager.revokeBestEffort(context.Background(), "https://reana.example.org", Metadata{}, "refresh"); warning != "" {
		t.Fatalf(
			"warning = %q, want empty when revocation endpoint is missing",
			warning,
		)
	}
	if warning := manager.revokeBestEffort(context.Background(), "https://reana.example.org", Metadata{
		RevocationEndpoint: "https://issuer.example.org/revoke",
	}, ""); warning != "" {
		t.Fatalf(
			"warning = %q, want empty when refresh token is missing",
			warning,
		)
	}
}

func TestRevokeBestEffortReturnsNetworkErrorMessage(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network unreachable")
	})
	warning := manager.revokeBestEffort(
		context.Background(),
		"https://reana.example.org",
		Metadata{
			RevocationEndpoint: "https://issuer.example.org/revoke",
		},
		"refresh",
	)
	if !strings.Contains(warning, "network unreachable") {
		t.Fatalf("warning = %q, want network error message", warning)
	}
}

func TestRevokeBestEffortReturnsHTTPStatusMessage(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusInternalServerError, ""), nil
	})
	warning := manager.revokeBestEffort(
		context.Background(),
		"https://reana.example.org",
		Metadata{
			RevocationEndpoint: "https://issuer.example.org/revoke",
		},
		"refresh",
	)
	if !strings.Contains(warning, "HTTP 500") {
		t.Fatalf("warning = %q, want HTTP 500 message", warning)
	}
}

func TestPostFormReturnsWrappedErrorOnTransportFailure(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset")
	})
	var target tokenResponse
	_, err := manager.postForm(
		context.Background(),
		"https://reana.example.org",
		"https://issuer.example.org/token",
		"token refresh",
		url.Values{},
		&target,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "Could not connect to") {
		t.Fatalf("expected wrapped transport error, got %v", err)
	}
}

func TestPostFormReturnsErrorOnNonJSONResponse(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, "not json"), nil
	})
	var target tokenResponse
	_, err := manager.postForm(
		context.Background(),
		"https://reana.example.org",
		"https://issuer.example.org/token",
		"token refresh",
		url.Values{},
		&target,
	)
	if err == nil || !strings.Contains(err.Error(), "non-JSON response") {
		t.Fatalf("expected non-JSON response error, got %v", err)
	}
}

func TestPostFormRejectsUnrepresentableTokenLifetimes(t *testing.T) {
	for name, payload := range map[string]string{
		"fractional": `{"access_token":"token","expires_in":1.5}`,
		"oversized":  `{"access_token":"token","expires_in":9223372036854775808}`,
	} {
		t.Run(name, func(t *testing.T) {
			manager := testManager(
				t,
				func(*http.Request) (*http.Response, error) {
					return jsonResponse(http.StatusOK, payload), nil
				},
			)
			var target tokenResponse
			_, err := manager.postForm(
				context.Background(),
				"https://reana.example.org",
				"https://issuer.example.org/token",
				"token refresh",
				url.Values{},
				&target,
			)
			if err == nil {
				t.Fatal("invalid lifetime unexpectedly accepted")
			}
		})
	}
}

func TestRefreshErrorsWhenNoRefreshToken(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected network call")
		return nil, nil
	})
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		Issuer:        "https://issuer.example.org",
		TokenEndpoint: "https://issuer.example.org/token",
		ClientID:      "reana-cli",
	}, true); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Refresh(
		context.Background(),
		"https://reana.example.org",
		Credentials{},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "Run `reana-client-go login --server") {
		t.Fatalf("expected missing refresh token error, got %v", err)
	}
}

func TestRefreshErrorsOnInvalidStoredMetadata(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected network call")
		return nil, nil
	})
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		RefreshToken: "refresh",
	}, true); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Refresh(
		context.Background(),
		"https://reana.example.org",
		Credentials{},
	)
	if !IsAuthenticationError(err) {
		t.Fatalf(
			"expected an AuthenticationError for invalid stored metadata, got %v",
			err,
		)
	}
}

func TestRefreshReturnsServerErrorMessage(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(
			http.StatusBadRequest,
			`{"error":"server_error","error_description":"issuer is unavailable"}`,
		), nil
	})
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		Issuer:        "https://issuer.example.org",
		TokenEndpoint: "https://issuer.example.org/token",
		ClientID:      "reana-cli",
		RefreshToken:  "refresh",
	}, true); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Refresh(
		context.Background(),
		"https://reana.example.org",
		Credentials{},
	)
	if err == nil || !strings.Contains(err.Error(), "issuer is unavailable") {
		t.Fatalf("expected server error message, got %v", err)
	}
}

func TestRefreshClearsCredentialsOnInvalidGrant(t *testing.T) {
	manager := testManager(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(
			http.StatusBadRequest,
			`{"error":"invalid_grant"}`,
		), nil
	})
	if _, err := manager.Store.Put("https://reana.example.org", Credentials{
		Issuer:        "https://issuer.example.org",
		TokenEndpoint: "https://issuer.example.org/token",
		ClientID:      "reana-cli",
		RefreshToken:  "refresh",
	}, true); err != nil {
		t.Fatal(err)
	}
	_, err := manager.Refresh(
		context.Background(),
		"https://reana.example.org",
		Credentials{},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "Run `reana-client-go login --server") {
		t.Fatalf("expected cleared-credentials login error, got %v", err)
	}
	stored, getErr := manager.Store.Get("https://reana.example.org")
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.RefreshToken != "" {
		t.Fatalf("refresh token was not cleared: %+v", stored)
	}
}

func TestLoginDeviceReturnsErrorForNonSuccessDeviceAuthorizationStatus(
	t *testing.T,
) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "device_authorization_endpoint":"https://issuer.example.org/device",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/device":
				return jsonResponse(
					http.StatusBadRequest,
					`{"error":"invalid_client","error_description":"unknown client"}`,
				), nil
			default:
				t.Fatalf("unexpected request: %s", request.URL)
				return nil, nil
			}
		},
	)
	_, err := manager.LoginDevice(
		context.Background(),
		"https://reana.example.org",
		func(DevicePrompt) {},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown client") {
		t.Fatalf("expected device authorization failure, got %v", err)
	}
}

func TestLoginDeviceReturnsErrorForIncompleteDeviceResponse(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "device_authorization_endpoint":"https://issuer.example.org/device",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/device":
				return jsonResponse(
					http.StatusOK,
					`{"user_code":"ABCD","verification_uri":"https://issuer.example.org/verify"}`,
				), nil
			default:
				t.Fatalf("unexpected request: %s", request.URL)
				return nil, nil
			}
		},
	)
	_, err := manager.LoginDevice(
		context.Background(),
		"https://reana.example.org",
		func(DevicePrompt) {},
	)
	if err == nil || !strings.Contains(err.Error(), "valid code and expiry") {
		t.Fatalf("expected incomplete device response error, got %v", err)
	}
}

func TestLoginDeviceRejectsUnusablePromptsAndDurations(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"missing prompt", `{"device_code":"device","expires_in":300}`},
		{
			"missing user code",
			`{"device_code":"device","verification_uri":"https://issuer.example.org/verify","expires_in":300}`,
		},
		{
			"insecure prompt",
			`{"device_code":"device","user_code":"ABCD","verification_uri":"http://issuer.example.org/verify","expires_in":300}`,
		},
		{
			"oversized expiry",
			`{"device_code":"device","verification_uri_complete":"https://issuer.example.org/verify?code=ABCD","expires_in":3601}`,
		},
		{
			"negative interval",
			`{"device_code":"device","user_code":"ABCD","verification_uri":"https://issuer.example.org/verify","expires_in":300,"interval":-1}`,
		},
		{
			"oversized interval",
			`{"device_code":"device","user_code":"ABCD","verification_uri":"https://issuer.example.org/verify","expires_in":300,"interval":3601}`,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			manager := testManager(
				t,
				func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case discoveryPath:
						return jsonResponse(http.StatusOK, `{
                        "issuer":"https://issuer.example.org",
                        "authorization_endpoint":"https://issuer.example.org/auth",
                        "token_endpoint":"https://issuer.example.org/token",
                        "device_authorization_endpoint":"https://issuer.example.org/device",
                        "reana_cli_client_id":"reana-cli"
                    }`), nil
					case "/device":
						return jsonResponse(
							http.StatusOK,
							testCase.payload,
						), nil
					default:
						t.Fatalf("unexpected request: %s", request.URL)
						return nil, nil
					}
				},
			)
			prompted := false
			_, err := manager.LoginDevice(
				context.Background(),
				"https://reana.example.org",
				func(DevicePrompt) { prompted = true },
			)
			if err == nil || prompted {
				t.Fatalf(
					"expected controlled rejection before display, got %v",
					err,
				)
			}
		})
	}
}

func TestLoginDeviceTerminalPollingErrors(t *testing.T) {
	cases := []struct {
		name       string
		pollBody   string
		wantErrMsg string
	}{
		{
			"expired token",
			`{"error":"expired_token"}`,
			"device login expired",
		},
		{
			"access denied",
			`{"error":"access_denied"}`,
			"device login was denied",
		},
		{
			"unknown error",
			`{"error":"temporarily_unavailable","error_description":"try later"}`,
			"try later",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := testManager(
				t,
				func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case discoveryPath:
						return jsonResponse(http.StatusOK, `{
                        "issuer":"https://issuer.example.org",
                        "authorization_endpoint":"https://issuer.example.org/auth",
                        "token_endpoint":"https://issuer.example.org/token",
                        "device_authorization_endpoint":"https://issuer.example.org/device",
                        "reana_cli_client_id":"reana-cli"
                    }`), nil
					case "/device":
						return jsonResponse(
							http.StatusOK,
							`{"device_code":"device","user_code":"ABCD","verification_uri":"https://issuer.example.org/verify","expires_in":300,"interval":1}`,
						), nil
					case "/token":
						return jsonResponse(
							http.StatusBadRequest,
							testCase.pollBody,
						), nil
					default:
						t.Fatalf("unexpected request: %s", request.URL)
						return nil, nil
					}
				},
			)
			_, err := manager.LoginDevice(
				context.Background(),
				"https://reana.example.org",
				func(DevicePrompt) {},
			)
			if err == nil ||
				!strings.Contains(err.Error(), testCase.wantErrMsg) {
				t.Fatalf(
					"error = %v, want it to contain %q",
					err,
					testCase.wantErrMsg,
				)
			}
		})
	}
}

func TestLoginDeviceHonoursSlowDownBeforeSucceeding(t *testing.T) {
	polls := 0
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "device_authorization_endpoint":"https://issuer.example.org/device",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/device":
				return jsonResponse(
					http.StatusOK,
					`{"device_code":"device","user_code":"ABCD","verification_uri":"https://issuer.example.org/verify","expires_in":300,"interval":1}`,
				), nil
			case "/token":
				polls++
				if polls == 1 {
					return jsonResponse(
						http.StatusBadRequest,
						`{"error":"slow_down"}`,
					), nil
				}
				return jsonResponse(
					http.StatusOK,
					`{"access_token":"a.b.c","refresh_token":"refresh","expires_in":3600}`,
				), nil
			default:
				t.Fatalf("unexpected request: %s", request.URL)
				return nil, nil
			}
		},
	)
	credentials, err := manager.LoginDevice(
		context.Background(),
		"https://reana.example.org",
		func(DevicePrompt) {},
	)
	if err != nil {
		t.Fatal(err)
	}
	if polls != 2 || credentials.AccessToken != "a.b.c" {
		t.Fatalf("polls = %d, credentials = %+v", polls, credentials)
	}
}

func TestLoginBrowserReturnsCallbackError(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
            "issuer":"https://issuer.example.org",
            "authorization_endpoint":"https://issuer.example.org/auth",
            "token_endpoint":"https://issuer.example.org/token",
            "reana_cli_client_id":"reana-cli"
        }`), nil
		},
	)
	_, err := manager.LoginBrowser(
		context.Background(),
		"https://reana.example.org",
		func(string) {},
		func(authorizationURL string) error {
			parsed, parseErr := url.Parse(authorizationURL)
			if parseErr != nil {
				return parseErr
			}
			redirect, parseErr := url.Parse(parsed.Query().Get("redirect_uri"))
			if parseErr != nil {
				return parseErr
			}
			callback := redirect.String() +
				"?error=access_denied&error_description=user+declined&state=" +
				url.QueryEscape(parsed.Query().Get("state"))
			response, requestErr := http.Get(callback) //nolint:gosec
			if requestErr != nil {
				return requestErr
			}
			defer response.Body.Close()
			body, readErr := io.ReadAll(response.Body)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(body), "login complete") {
				t.Error(
					"callback page must not claim success for an error response",
				)
			}
			if !strings.Contains(string(body), "login failed") {
				t.Errorf("expected a failure page, got %q", body)
			}
			return nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "user declined") {
		t.Fatalf("expected callback error to propagate, got %v", err)
	}
}

func TestLoginBrowserRejectsStateMismatch(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
            "issuer":"https://issuer.example.org",
            "authorization_endpoint":"https://issuer.example.org/auth",
            "token_endpoint":"https://issuer.example.org/token",
            "reana_cli_client_id":"reana-cli"
        }`), nil
		},
	)
	_, err := manager.LoginBrowser(
		context.Background(),
		"https://reana.example.org",
		func(string) {},
		func(authorizationURL string) error {
			parsed, parseErr := url.Parse(authorizationURL)
			if parseErr != nil {
				return parseErr
			}
			redirect, parseErr := url.Parse(parsed.Query().Get("redirect_uri"))
			if parseErr != nil {
				return parseErr
			}
			callback := redirect.String() + "?code=code&state=wrong-state"
			response, requestErr := http.Get(callback) //nolint:gosec
			if requestErr == nil {
				response.Body.Close()
			}
			return requestErr
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "state parameter mismatch") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
}

func TestLoginBrowserRejectsTokenExchangeFailure(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case discoveryPath:
				return jsonResponse(http.StatusOK, `{
                "issuer":"https://issuer.example.org",
                "authorization_endpoint":"https://issuer.example.org/auth",
                "token_endpoint":"https://issuer.example.org/token",
                "reana_cli_client_id":"reana-cli"
            }`), nil
			case "/token":
				return jsonResponse(
					http.StatusBadRequest,
					`{"error":"invalid_grant","error_description":"code expired"}`,
				), nil
			default:
				t.Fatalf("unexpected request: %s", request.URL)
				return nil, nil
			}
		},
	)
	_, err := manager.LoginBrowser(
		context.Background(),
		"https://reana.example.org",
		func(string) {},
		func(authorizationURL string) error {
			parsed, parseErr := url.Parse(authorizationURL)
			if parseErr != nil {
				return parseErr
			}
			callback := parsed.Query().
				Get("redirect_uri") +
				"?code=code&state=" + url.QueryEscape(
				parsed.Query().Get("state"),
			)
			response, requestErr := http.Get(callback) //nolint:gosec
			if requestErr == nil {
				response.Body.Close()
			}
			return requestErr
		},
	)
	if err == nil || !strings.Contains(err.Error(), "code expired") {
		t.Fatalf("expected token exchange failure, got %v", err)
	}
}

func TestLoginBrowserTimesOutWhenNoCallbackArrives(t *testing.T) {
	manager := testManager(
		t,
		func(request *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
            "issuer":"https://issuer.example.org",
            "authorization_endpoint":"https://issuer.example.org/auth",
            "token_endpoint":"https://issuer.example.org/token",
            "reana_cli_client_id":"reana-cli"
        }`), nil
		},
	)
	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Millisecond,
	)
	defer cancel()
	_, err := manager.LoginBrowser(
		ctx,
		"https://reana.example.org",
		func(string) {},
		func(string) error { return nil },
	)
	if err == nil || !strings.Contains(err.Error(), "browser login timed out") {
		t.Fatalf("expected browser login timeout, got %v", err)
	}
}

func TestFirstNonEmptyReturnsFirstNonEmptyValue(t *testing.T) {
	cases := []struct {
		name   string
		values []string
		want   string
	}{
		{"first wins", []string{"a", "b"}, "a"},
		{"skips leading empties", []string{"", "", "b"}, "b"},
		{"all empty falls back", []string{"", ""}, "unknown error"},
		{"no arguments falls back", nil, "unknown error"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := firstNonEmpty(testCase.values...); got != testCase.want {
				t.Fatalf(
					"firstNonEmpty(%v) = %q, want %q",
					testCase.values,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestIsJWTRecognisesCompactSerialisation(t *testing.T) {
	cases := []struct {
		name  string
		token string
		want  bool
	}{
		{"three non-empty parts", "header.payload.signature", true},
		{"unsigned JWT (empty signature)", "header.payload.", false},
		{"missing a part", "header.payload", false},
		{"too many parts", "a.b.c.d", false},
		{"empty string", "", false},
		{"opaque token", "not-a-jwt-at-all", false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := IsJWT(testCase.token); got != testCase.want {
				t.Fatalf(
					"IsJWT(%q) = %v, want %v",
					testCase.token,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestIsAuthenticationErrorDetectsWrappedAuthenticationErrors(t *testing.T) {
	authErr := authenticationError("not signed in")
	if !IsAuthenticationError(authErr) {
		t.Fatal("expected the raw AuthenticationError to be detected")
	}
	wrapped := fmt.Errorf("during login: %w", authErr)
	if !IsAuthenticationError(wrapped) {
		t.Fatal("expected a wrapped AuthenticationError to be detected")
	}
	if IsAuthenticationError(errors.New("plain error")) {
		t.Fatal("a plain error must not be reported as an AuthenticationError")
	}
	if IsAuthenticationError(nil) {
		t.Fatal("a nil error must not be reported as an AuthenticationError")
	}
}

func readForm(t *testing.T, request *http.Request) url.Values {
	t.Helper()
	contents, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	form, err := url.ParseQuery(string(contents))
	if err != nil {
		t.Fatal(err)
	}
	return form
}
