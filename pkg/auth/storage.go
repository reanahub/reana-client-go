/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Credentials is one server's OIDC metadata and token material.
type Credentials struct {
	TLS   *TLSSettings `json:"tls,omitempty"`
	extra map[string]json.RawMessage

	Issuer                      string `json:"issuer,omitempty"`
	ClientID                    string `json:"client_id,omitempty"`
	TokenEndpoint               string `json:"token_endpoint,omitempty"`
	AuthorizationEndpoint       string `json:"authorization_endpoint,omitempty"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint,omitempty"`
	RevocationEndpoint          string `json:"revocation_endpoint,omitempty"`
	AccessToken                 string `json:"access_token,omitempty"`
	AccessTokenExpiresAt        string `json:"access_token_expires_at,omitempty"`
	RefreshToken                string `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt       string `json:"refresh_token_expires_at,omitempty"`
	CredentialEpoch             int64  `json:"credential_epoch,omitempty"`
}

type credentialConfig struct {
	extra map[string]json.RawMessage

	ActiveServer string                 `json:"active_server"`
	Servers      map[string]Credentials `json:"servers"`
}

// Store provides atomic, cross-process-safe access to the credential file.
type Store struct {
	Path string
}

// NewStore opens the default shared credential-store location.
func NewStore() (*Store, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	defaultPath, err := defaultConfigPath()
	if err != nil {
		return nil, err
	}
	if path == defaultPath {
		if err := os.Chmod(directory, 0o700); err != nil {
			return nil, err
		}
	}
	return &Store{Path: path}, nil
}

func emptyCredentialConfig() credentialConfig {
	return credentialConfig{Servers: make(map[string]Credentials)}
}

func (s *Store) loadUnlocked() (credentialConfig, error) {
	contents, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyCredentialConfig(), nil
	}
	if err != nil {
		return credentialConfig{}, err
	}
	config := emptyCredentialConfig()
	if err := json.Unmarshal(contents, &config); err != nil {
		return credentialConfig{}, fmt.Errorf(
			"invalid REANA client credential file: %w",
			err,
		)
	}
	if config.Servers == nil {
		config.Servers = make(map[string]Credentials)
	}
	return config, nil
}

func (s *Store) saveUnlocked(config credentialConfig) error {
	directory := filepath.Dir(s.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')
	temporary, err := os.CreateTemp(directory, ".reana-client-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, s.Path); err != nil {
		return err
	}
	return os.Chmod(s.Path, 0o600)
}

func (s *Store) withLock(update func(*credentialConfig) error) error {
	lock, err := acquireLock(s.Path+".lock", true)
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	config, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if err := update(&config); err != nil {
		return err
	}
	return s.saveUnlocked(config)
}

// ActiveServer returns the saved active server.
func (s *Store) ActiveServer() (string, error) {
	if err := CheckRetiredEnvironment(); err != nil {
		return "", err
	}
	lock, err := acquireLock(s.Path+".lock", true)
	if err != nil {
		return "", err
	}
	defer releaseLock(lock)
	config, err := s.loadUnlocked()
	if err != nil || config.ActiveServer == "" {
		return config.ActiveServer, err
	}
	return NormalizeServerURL(config.ActiveServer)
}

// Get returns the credential entry for serverURL.
func (s *Store) Get(serverURL string) (Credentials, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return Credentials{}, err
	}
	lock, err := acquireLock(s.Path+".lock", true)
	if err != nil {
		return Credentials{}, err
	}
	defer releaseLock(lock)
	config, err := s.loadUnlocked()
	if err != nil {
		return Credentials{}, err
	}
	return config.Servers[normalized], nil
}

// put stores the entry built from the previous one and increments its
// concurrency epoch. A non-nil epoch makes the write conditional on it.
func (s *Store) put(
	serverURL string,
	epoch *int64,
	makeActive bool,
	build func(previous Credentials) Credentials,
) (Credentials, bool, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return Credentials{}, false, err
	}
	var stored Credentials
	matched := false
	err = s.withLock(func(config *credentialConfig) error {
		previous := config.Servers[normalized]
		if epoch != nil && previous.CredentialEpoch != *epoch {
			return nil
		}
		entry := build(previous)
		entry.CredentialEpoch = previous.CredentialEpoch + 1
		config.Servers[normalized] = entry
		if makeActive {
			config.ActiveServer = normalized
		}
		stored, matched = entry, true
		return nil
	})
	return stored, matched, err
}

// Put updates a server entry and increments its concurrency epoch.
func (s *Store) Put(
	serverURL string,
	entry Credentials,
	makeActive bool,
) (Credentials, error) {
	stored, _, err := s.put(
		serverURL,
		nil,
		makeActive,
		func(previous Credentials) Credentials {
			return preserveSettings(entry, previous)
		},
	)
	return stored, err
}

// PutIfEpoch stores refreshed credentials only if no login/logout raced it.
func (s *Store) PutIfEpoch(
	serverURL string,
	entry Credentials,
	epoch int64,
) (Credentials, bool, error) {
	return s.put(
		serverURL,
		&epoch,
		false,
		func(previous Credentials) Credentials {
			return preserveSettings(entry, previous)
		},
	)
}

// PutRecovery stores a rotated refresh token from a rejected token response
// without replacing the rest of the entry or changing the active server.
func (s *Store) PutRecovery(
	serverURL string,
	recovery Credentials,
) (Credentials, error) {
	stored, _, err := s.put(
		serverURL,
		nil,
		false,
		func(previous Credentials) Credentials {
			return mergeRecovery(recovery, previous)
		},
	)
	return stored, err
}

// PutRecoveryIfEpoch is PutRecovery, unless a login/logout raced it.
func (s *Store) PutRecoveryIfEpoch(
	serverURL string,
	recovery Credentials,
	epoch int64,
) (Credentials, bool, error) {
	return s.put(
		serverURL,
		&epoch,
		false,
		func(previous Credentials) Credentials {
			return mergeRecovery(recovery, previous)
		},
	)
}

// ClearTokens removes tokens while preserving discovery metadata.
func (s *Store) ClearTokens(
	serverURL string,
	onlyRefreshToken string,
) (bool, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return false, err
	}
	cleared := false
	err = s.withLock(func(config *credentialConfig) error {
		entry := config.Servers[normalized]
		if onlyRefreshToken != "" && entry.RefreshToken != onlyRefreshToken {
			return nil
		}
		entry.AccessToken = ""
		entry.AccessTokenExpiresAt = ""
		entry.RefreshToken = ""
		entry.RefreshTokenExpiresAt = ""
		entry.CredentialEpoch++
		config.Servers[normalized] = entry
		cleared = true
		return nil
	})
	return cleared, err
}

// Logout holds the credential lock across remote revocation and local clearing.
func (s *Store) Logout(
	serverURL string,
	revoke func(Credentials) string,
) (string, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return "", err
	}
	var warning string
	err = s.withLock(func(config *credentialConfig) error {
		entry := config.Servers[normalized]
		warning = revoke(entry)
		entry.AccessToken = ""
		entry.AccessTokenExpiresAt = ""
		entry.RefreshToken = ""
		entry.RefreshTokenExpiresAt = ""
		entry.CredentialEpoch++
		config.Servers[normalized] = entry
		return nil
	})
	return warning, err
}

func (s *Store) refreshLockPath(serverURL string) (string, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(normalized))
	return s.Path + ".refresh." + hex.EncodeToString(hash[:8]) + ".lock", nil
}

func (s *Store) tryRefreshLock(serverURL string) (*os.File, error) {
	path, err := s.refreshLockPath(serverURL)
	if err != nil {
		return nil, err
	}
	return acquireLock(path, false)
}

func (s *Store) waitRefreshLock(
	serverURL string,
	timeout time.Duration,
) (bool, error) {
	path, err := s.refreshLockPath(serverURL)
	if err != nil {
		return false, err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		lock, err := acquireLock(path, false)
		if err != nil {
			return false, err
		}
		if lock != nil {
			releaseLock(lock)
			return true, nil
		}
		time.Sleep(lockPollInterval)
	}
	return false, nil
}
