/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRemoveServerRevocation(t *testing.T) {
	for _, status := range []int{200, 307, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
			t.Setenv("REANA_SERVER_CA_CERTS", "")
			calls := 0
			server := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.FormValue("token") != "refresh-secret" ||
						r.FormValue("client_id") != "reana-client" {
						t.Error("wrong revocation form")
					}
					if status == 307 {
						w.Header().
							Set("Location", "https://iam.example/revoke?secret=redirect")
					}
					w.WriteHeader(status)
				}),
			)
			defer server.Close()
			manager, err := NewManager()
			if err != nil {
				t.Fatal(err)
			}
			disabled := false
			if _, err := manager.Store.Put(server.URL, Credentials{
				RefreshToken: "refresh-secret", ClientID: "reana-client", RevocationEndpoint: server.URL + "/revoke",
				TLS: &TLSSettings{Verify: &disabled},
			}, true); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(manager.Store.Path)
			_, err = manager.RemoveServer(
				context.Background(),
				server.URL,
				false,
			)
			if calls != 1 {
				t.Fatalf("revocations: %d", calls)
			}
			if status != 200 {
				if err == nil ||
					!strings.Contains(err.Error(), "--local-only") ||
					!strings.Contains(
						err.Error(),
						fmt.Sprintf("HTTP %d", status),
					) ||
					!strings.Contains(
						err.Error(),
						server.URL+" (from server-remove argument)",
					) {
					t.Fatalf("expected failure: %v", err)
				}
				if strings.Contains(err.Error(), "secret") {
					t.Fatalf("request data exposed: %v", err)
				}
				if status == 307 &&
					!strings.Contains(
						err.Error(),
						"Refusing to follow a redirect",
					) {
					t.Fatalf("missing redirect explanation: %v", err)
				}
				after, _ := os.ReadFile(manager.Store.Path)
				if string(before) != string(after) {
					t.Fatal("failure changed saved state")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				records, _ := manager.Store.ListServers()
				active, _ := manager.Store.ActiveServer()
				if len(records) != 0 || active != "" {
					t.Fatal("record not removed")
				}
			}
		})
	}
}

func TestRemoveServerExplainsTransportFailures(t *testing.T) {
	for _, failure := range []string{"certificate", "external issuer", "refused"} {
		t.Run(failure, func(t *testing.T) {
			t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
			t.Setenv("REANA_SERVER_CA_CERTS", "")
			endpoint := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Error("untrusted endpoint received credentials")
				}),
			)
			defer endpoint.Close()
			if failure == "refused" {
				endpoint.Close()
			}
			target := endpoint.URL
			verify := true
			if failure == "external issuer" {
				target = "https://saved.example.org"
				verify = false
			}
			manager, err := NewManager()
			if err != nil {
				t.Fatal(err)
			}
			_, err = manager.Store.Put(target, Credentials{
				RefreshToken: "refresh-secret", ClientID: "cli",
				RevocationEndpoint: endpoint.URL + "/revoke?secret=endpoint",
				TLS:                &TLSSettings{Verify: &verify},
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = manager.Store.Put("https://selected.example", Credentials{}, true); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(manager.Store.Path)
			_, err = manager.RemoveServer(context.Background(), target, false)
			if err == nil {
				t.Fatal("revocation unexpectedly succeeded")
			}
			want := "certificate is not trusted"
			if failure == "refused" {
				want = "connection was refused"
			}
			for _, part := range []string{want, target + " (from server-remove argument)", "Resolve the reported problem before retrying", "--local-only"} {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("missing %q: %v", part, err)
				}
			}
			if strings.Contains(
				err.Error(),
				"--no-tls-verify",
			) != (failure == "certificate") {
				t.Fatalf("incorrect bypass guidance: %v", err)
			}
			if failure == "external issuer" &&
				!strings.Contains(
					err.Error(),
					"does not apply to this identity provider",
				) {
				t.Fatalf("missing external issuer explanation: %v", err)
			}
			for _, hidden := range []string{"selected.example", "refresh-secret", "secret=endpoint"} {
				if strings.Contains(err.Error(), hidden) {
					t.Fatalf("unrelated or secret details: %v", err)
				}
			}
			after, _ := os.ReadFile(manager.Store.Path)
			if string(before) != string(after) {
				t.Fatal("failed revocation changed store")
			}
		})
	}
}

func TestRemoveServerMissingMetadata(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Store.Put("a.example", Credentials{RefreshToken: "secret"}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RemoveServer(context.Background(), "a.example", false); err == nil {
		t.Fatal("missing metadata accepted")
	}
	entry, _ := manager.Store.Get("a.example")
	if entry.RefreshToken != "secret" {
		t.Fatal("token removed on failure")
	}
	if _, err := manager.RemoveServer(context.Background(), "a.example", true); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveServerWaitsForRotation(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	revoked := make(chan string, 1)
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			revoked <- r.FormValue("token")
			w.WriteHeader(200)
		}),
	)
	defer server.Close()
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	entry, err := manager.Store.Put(server.URL, Credentials{
		RefreshToken: "old", ClientID: "reana-client", RevocationEndpoint: server.URL + "/revoke",
		TLS: &TLSSettings{Verify: &disabled},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := manager.Store.tryRefreshLock(server.URL)
	if err != nil || lock == nil {
		t.Fatalf("lock: %v", err)
	}
	done := make(chan error, 1)
	go func() { _, err := manager.RemoveServer(context.Background(), server.URL, false); done <- err }()
	select {
	case err := <-done:
		releaseLock(lock)
		t.Fatalf("removed during refresh: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	entry.RefreshToken = "rotated"
	if _, err := manager.Store.Put(server.URL, entry, false); err != nil {
		releaseLock(lock)
		t.Fatal(err)
	}
	releaseLock(lock)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("removal did not complete")
	}
	if token := <-revoked; token != "rotated" {
		t.Fatalf("revoked %q", token)
	}
	// A refresh entering after removal must re-read disk, never reuse its input.
	if _, err := manager.Refresh(context.Background(), server.URL, entry); err == nil {
		t.Fatal("removed credentials refreshed")
	}
}

func TestSwitchKeepsRefreshAndUnknownFields(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"active_server":"https://a.example","future":"keep","servers":{"https://a.example":{"refresh_token":"old","credential_epoch":1,"future":"keep","tls":{"verify":false,"future":"keep"}},"https://b.example":{}}}`
	if err := os.WriteFile(store.Path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	entry, _ := store.Get("a.example")
	if _, err := store.UseServer("b.example"); err != nil {
		t.Fatal(err)
	}
	entry.RefreshToken = "new"
	if _, matched, err := store.PutIfEpoch("a.example", entry, entry.CredentialEpoch); err != nil ||
		!matched {
		t.Fatalf("refresh: %v, %v", matched, err)
	}
	active, _ := store.ActiveServer()
	if active != "https://b.example" {
		t.Fatal("refresh reverted selection")
	}
	data, _ := os.ReadFile(store.Path)
	if strings.Count(string(data), `"future"`) != 3 {
		t.Fatal("unknown fields lost")
	}
}

func TestListServersWithBrokenTLSPolicy(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"active_server":null,"servers":{"https://a.example":{"tls":{"verify":"bad"}},"https://b.example":{"tls":{"verify":false}},"https://c.example":{}}}`
	if err := os.WriteFile(store.Path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	records, err := store.ListServers()
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"invalid", "disabled", "enabled"} {
		if records[i].TLSVerification != want {
			t.Fatalf("record %d: %v", i, records[i])
		}
	}
	after, _ := os.ReadFile(store.Path)
	if string(after) != payload {
		t.Fatal("listing changed store")
	}
}

func TestRemovalRevocationIgnoresRetiredExports(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_URL", "https://wrong.example")
	t.Setenv("REANA_SERVER_TLS_VERIFY", "1")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	calls := 0
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.FormValue("token") != "secret" {
				t.Error("wrong token")
			}
			w.WriteHeader(200)
		}),
	)
	defer server.Close()
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	_, err = manager.Store.Put(
		server.URL,
		Credentials{
			RefreshToken:       "secret",
			ClientID:           "reana-client",
			RevocationEndpoint: server.URL + "/revoke",
			TLS:                &TLSSettings{Verify: &disabled},
		},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.RemoveServer(context.Background(), server.URL, false)
	if err != nil || calls != 1 || result.Unrevoked {
		t.Fatalf("remove: %v, calls %d, %v", result, calls, err)
	}
}

func TestLocalRemovalReportsOnlyUnrevokedTokens(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Store.Put(
		"a.example",
		Credentials{RefreshToken: "secret"},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.RemoveServer(context.Background(), "a.example", true)
	if err != nil || !result.Unrevoked {
		t.Fatalf("remove: %v %v", result, err)
	}
}

func TestRemovalKeepsExternalIssuerVerified(t *testing.T) {
	t.Setenv("REANA_CLIENT_CONFIG", t.TempDir()+"/credentials.json")
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	calls := 0
	issuer := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(200)
		}),
	)
	defer issuer.Close()
	manager, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	_, err = manager.Store.Put("https://reana.example", Credentials{
		RefreshToken: "secret", ClientID: "reana-client",
		RevocationEndpoint: issuer.URL + "/revoke", TLS: &TLSSettings{Verify: &disabled},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(manager.Store.Path)
	if _, err := manager.RemoveServer(context.Background(), "https://reana.example", false); err == nil {
		t.Fatal("REANA bypass applied to external issuer")
	}
	after, _ := os.ReadFile(manager.Store.Path)
	if calls != 0 || string(before) != string(after) {
		t.Fatal("untrusted issuer received tokens or removal changed store")
	}
}
