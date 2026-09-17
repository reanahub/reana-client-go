/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package auth implements OIDC authentication for the REANA command-line client.
package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	configPathEnv        = "REANA_CLIENT_CONFIG"
	caCertsEnv           = "REANA_SERVER_CA_CERTS"
	tlsVerifyEnv         = "REANA_SERVER_TLS_VERIFY"
	loginLoopbackPortEnv = "REANA_CLIENT_LOGIN_LOOPBACK_PORT"
)

var tlsWarningOnce sync.Once

// NormalizeServerURL returns the canonical credential-store key for a server.
func NormalizeServerURL(serverURL string) (string, error) {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return "", errors.New("REANA server URL is not set")
	}
	if !strings.Contains(serverURL, "://") {
		serverURL = "https://" + serverURL
	}
	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("REANA server URL must include scheme and host")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	// DNS hostnames are case-insensitive; without this, two otherwise-
	// identical server URLs differing only in host casing would normalize
	// to different credential-store keys instead of being recognized as
	// the same server.
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Scheme != "https" {
		host := parsed.Hostname()
		ip := net.ParseIP(host)
		if parsed.Scheme != "http" || (!strings.EqualFold(host, "localhost") &&
			(ip == nil || !ip.IsLoopback())) {
			return "", errors.New("REANA server URL must use HTTPS")
		}
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New(
			"REANA server URL must not contain credentials, a query, or a fragment",
		)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

// ConfigPath returns the shared Python/Go client credential-store path.
func ConfigPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(configPathEnv)); configured != "" {
		if strings.HasPrefix(configured, "~"+string(filepath.Separator)) {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configured = filepath.Join(
				home,
				strings.TrimPrefix(configured, "~"+string(filepath.Separator)),
			)
		}
		return filepath.Abs(configured)
	}
	return defaultConfigPath()
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "reana", "reana-client.json"), nil
}

// NewHTTPClient builds an HTTP client using the REANA server TLS settings.
func NewHTTPClient(serverURLs ...string) (*http.Client, error) {
	serverURL := ""
	if len(serverURLs) > 0 {
		serverURL = serverURLs[0]
	}
	client, err := serverHTTPClient(serverURL, nil)
	if err == nil {
		warnTLSVerification(client, serverURL)
	}
	return client, err
}

// TLSVerify resolves the effective policy after applying a CA bundle override.
func TLSVerify(serverURL string, explicit *bool) (bool, error) {
	if err := CheckRetiredEnvironment(); err != nil {
		return false, err
	}
	if os.Getenv(caCertsEnv) != "" {
		if explicit != nil && !*explicit {
			return false, authenticationError(
				"--no-tls-verify conflicts with REANA_SERVER_CA_CERTS. Unset the CA bundle override first.",
			)
		}
		return true, nil
	}
	if explicit != nil {
		return *explicit, nil
	}
	if serverURL == "" {
		return true, nil
	}
	store, err := NewStore()
	if err != nil {
		return false, err
	}
	entry, err := store.Get(serverURL)
	if err != nil {
		return false, err
	}
	if entry.TLS != nil && entry.TLS.Verify != nil {
		return *entry.TLS.Verify, nil
	}
	return true, nil
}

func serverHTTPClient(serverURL string, explicit *bool) (*http.Client, error) {
	verify, err := TLSVerify(serverURL, explicit)
	if err != nil {
		return nil, err
	}
	return newHTTPClient(verify)
}

// effectiveTLSVerify binds the saved policy to this manager's invocation.
func (m *Manager) effectiveTLSVerify(serverURL string) (bool, error) {
	m.tlsMutex.Lock()
	defer m.tlsMutex.Unlock()
	if verify, ok := m.tlsPolicies[serverURL]; ok {
		return verify, nil
	}
	verify, err := TLSVerify(serverURL, m.TLSVerify)
	if err != nil {
		return false, err
	}
	if m.tlsPolicies == nil {
		m.tlsPolicies = make(map[string]bool)
	}
	m.tlsPolicies[serverURL] = verify
	return verify, nil
}

func (m *Manager) serverHTTPClient(serverURL string) (*http.Client, error) {
	verify, err := m.effectiveTLSVerify(serverURL)
	if err != nil {
		return nil, err
	}
	return newHTTPClient(verify)
}

// NewHTTPClient builds an API transport using this invocation's resolved policy.
func (m *Manager) NewHTTPClient(serverURL string) (*http.Client, error) {
	client, err := m.serverHTTPClient(serverURL)
	if err == nil {
		warnTLSVerification(client, serverURL)
	}
	return client, err
}

// TLSStatus reports the same policy used by this manager's transports.
func (m *Manager) TLSStatus(serverURL string) (string, error) {
	verify, err := m.effectiveTLSVerify(serverURL)
	if err != nil {
		return "", err
	}
	return tlsStatus(verify), nil
}

// TLSStatus describes effective verification, including CA overrides.
func TLSStatus(serverURL string, explicit *bool) (string, error) {
	verify, err := TLSVerify(serverURL, explicit)
	if err != nil {
		return "", err
	}
	return tlsStatus(verify), nil
}

func tlsStatus(verify bool) string {
	if !verify {
		return "disabled"
	}
	return "enabled"
}

// ResetTLSWarning starts a new CLI invocation's warning lifetime.
func ResetTLSWarning() { tlsWarningOnce = sync.Once{} }

// NewStrictHTTPClient verifies identity-provider certificates regardless of
// saved REANA bypass settings, while still trusting REANA_SERVER_CA_CERTS.
func NewStrictHTTPClient() (*http.Client, error) {
	if err := CheckRetiredEnvironment(); err != nil {
		return nil, err
	}
	return newHTTPClient(true)
}

// sameHTTPSOrigin compares hosts and effective ports without trusting DNS aliases.
func sameHTTPSOrigin(serverURL, endpoint string) bool {
	server, err := url.Parse(serverURL)
	if err != nil {
		return false
	}
	target, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	ports := [2]int{}
	for i, parsed := range []*url.URL{server, target} {
		if parsed.Scheme != "https" || parsed.Hostname() == "" ||
			parsed.User != nil {
			return false
		}
		ports[i] = 443
		if raw := parsed.Port(); raw != "" {
			port, err := strconv.Atoi(raw)
			if err != nil || port < 1 || port > 65535 {
				return false
			}
			ports[i] = port
		}
	}
	return strings.EqualFold(server.Hostname(), target.Hostname()) &&
		ports[0] == ports[1]
}

func newHTTPClient(verify bool) (*http.Client, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if caPath := strings.TrimSpace(os.Getenv(caCertsEnv)); caPath != "" {
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		pem, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("could not read REANA CA bundle: %w", err)
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New(
				"REANA CA bundle does not contain a valid certificate",
			)
		}
		tlsConfig.RootCAs = roots
	} else {
		tlsConfig.InsecureSkipVerify = !verify //nolint:gosec // Explicit per-server choice.
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func warnTLSVerification(client *http.Client, serverURL string) {
	transport, ok := client.Transport.(*http.Transport)
	if ok && transport.TLSClientConfig != nil &&
		transport.TLSClientConfig.InsecureSkipVerify {
		tlsWarningOnce.Do(
			func() { log.Warnf("TLS certificate verification is disabled for %s.", serverURL) },
		)
	}
}

// CheckRetiredEnvironment rejects stale exports before using a saved destination.
func CheckRetiredEnvironment() error {
	names := []string{}
	for _, name := range []string{"REANA_SERVER_URL", tlsVerifyEnv} {
		if os.Getenv(name) != "" {
			names = append(names, name)
		}
	}
	if len(names) != 0 {
		return authenticationError(
			"%s are no longer client inputs. Unset them, then run `reana-client-go login --server https://your-reana-server`. For self-signed development HTTPS, add --no-tls-verify.",
			strings.Join(names, " and "),
		)
	}
	return nil
}
