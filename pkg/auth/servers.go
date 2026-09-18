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
	"errors"
	"os"
	"sort"
)

// SavedServer exposes only connection settings, never credential material.
type SavedServer struct {
	URL             string
	Active          bool
	TLSVerification string
}

// AddServer saves a new connection without selecting or authenticating it.
func (s *Store) AddServer(serverURL string, verify *bool) (string, error) {
	server, err := NormalizeServerURL(serverURL)
	if err != nil {
		return "", err
	}
	err = s.withLock(func(config *credentialConfig) error {
		if _, exists := config.Servers[server]; exists {
			return authenticationError(
				"REANA server %s is already saved. Use `login --server` to update its TLS settings.",
				server,
			)
		}
		if _, err := resolveTLSVerify(nil, verify); err != nil {
			return err
		}
		entry := Credentials{}
		if verify != nil {
			entry.TLS = &TLSSettings{Verify: verify}
		}
		config.Servers[server] = entry
		return nil
	})
	return server, err
}

// ListServers returns a sorted, public view of one configuration snapshot.
func (s *Store) ListServers() ([]SavedServer, error) {
	lock, err := acquireLock(s.Path+".lock", true)
	if err != nil {
		return nil, err
	}
	defer releaseLock(lock)
	contents, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var config struct {
		ActiveServer string                     `json:"active_server"`
		Servers      map[string]json.RawMessage `json:"servers"`
	}
	if err := json.Unmarshal(contents, &config); err != nil {
		return nil, err
	}
	servers := make([]SavedServer, 0, len(config.Servers))
	for server, raw := range config.Servers {
		status := "invalid"
		var entry struct {
			TLS *TLSSettings `json:"tls"`
		}
		if err := json.Unmarshal(raw, &entry); err == nil {
			if verify, err := resolveTLSVerify(entry.TLS, nil); err == nil {
				status = tlsStatus(verify)
			}
		}
		servers = append(
			servers,
			SavedServer{
				URL:             server,
				Active:          server == config.ActiveServer,
				TLSVerification: status,
			},
		)
	}
	sort.Slice(
		servers,
		func(i, j int) bool { return servers[i].URL < servers[j].URL },
	)
	return servers, nil
}

func requireSavedServer(config *credentialConfig, server string) error {
	if _, exists := config.Servers[server]; !exists {
		return authenticationError(
			"No saved REANA server %s. Run `reana-client-go login --server %s` or `reana-client-go server-add %s`",
			server,
			server,
			server,
		)
	}
	return nil
}

// UseServer changes only the current selection, without network access.
func (s *Store) UseServer(serverURL string) (string, error) {
	server, err := NormalizeServerURL(serverURL)
	if err != nil {
		return "", err
	}
	err = s.withLock(func(config *credentialConfig) error {
		if err := requireSavedServer(config, server); err != nil {
			return err
		}
		config.ActiveServer = server
		return nil
	})
	return server, err
}

// RemovedServer reports whether local removal left remote credentials unrevoked.
type RemovedServer struct {
	URL       string
	Unrevoked bool
}

// RemoveServer revokes then forgets a record, preserving it on failure.
func (m *Manager) RemoveServer(
	ctx context.Context,
	serverURL string,
	localOnly bool,
) (RemovedServer, error) {
	server, err := NormalizeServerURL(serverURL)
	if err != nil {
		return RemovedServer{}, err
	}
	// Use the refresh lock before the store lock, matching token rotation's
	// order, so an in-flight refresh cannot recreate a removed record.
	path, err := m.Store.refreshLockPath(server)
	if err != nil {
		return RemovedServer{}, err
	}
	lock, err := acquireLock(path, true)
	if err != nil {
		return RemovedServer{}, err
	}
	defer releaseLock(lock)
	unrevoked := false
	err = m.Store.withLock(func(config *credentialConfig) error {
		if err := requireSavedServer(config, server); err != nil {
			return err
		}
		entry := config.Servers[server]
		unrevoked = localOnly && entry.RefreshToken != ""
		if !localOnly && entry.RefreshToken != "" {
			if entry.RevocationEndpoint == "" || entry.ClientID == "" {
				return authenticationError(
					"Cannot revoke saved credentials: revocation metadata is missing. Use --local-only to remove the record without revocation",
				)
			}
			if err := validateOIDCURL("revocation_endpoint", entry.RevocationEndpoint, true); err != nil {
				return err
			}
			// Construct directly from the locked entry; no manager configuration
			// or store access is allowed in the revocation request path.
			verify := true
			if sameHTTPSOrigin(server, entry.RevocationEndpoint) {
				verify, err = resolveTLSVerify(entry.TLS, nil)
				if err != nil {
					return err
				}
			}
			client, err := newHTTPClient(verify)
			if err != nil {
				return err
			}
			defer client.CloseIdleConnections()
			warnTLSVerification(client, server)
			if warning := revokeWithClient(ctx, client, server, metadataFromCredentials(entry), entry.RefreshToken); warning != "" {
				return ForServer(
					server, "server-remove argument",
					authenticationError(
						"Could not revoke saved credentials; the server was not removed.\n%s\nResolve the reported problem before retrying, or use --local-only to remove the record without revocation.",
						warning,
					),
				)
			}
		}
		delete(config.Servers, server)
		if config.ActiveServer == server {
			config.ActiveServer = ""
		}
		return nil
	})
	return RemovedServer{URL: server, Unrevoked: unrevoked}, err
}
