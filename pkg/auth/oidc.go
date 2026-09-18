/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	discoveryPath       = "/api/.well-known/openid-configuration"
	defaultScopes       = "openid profile email offline_access"
	expiryLeeway        = 60 * time.Second
	refreshLockWait     = 35 * time.Second
	deviceFlowMax       = time.Hour
	loopbackCallbackURL = "/callback"
)

// AuthenticationError is an authentication failure suitable for CLI output.
type AuthenticationError struct {
	Message string
	Cause   error
}

func (e *AuthenticationError) Error() string { return e.Message }

// Unwrap preserves typed transport causes for callers.
func (e *AuthenticationError) Unwrap() error { return e.Cause }

// Metadata is the authentication configuration relayed by REANA Server.
type Metadata struct {
	Issuer                      string `json:"issuer"`
	AuthorizationEndpoint       string `json:"authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	RevocationEndpoint          string `json:"revocation_endpoint"`
	CLIClientID                 string `json:"reana_cli_client_id"`
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        *int64 `json:"expires_in"`
	RefreshExpiresIn *int64 `json:"refresh_expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// DevicePrompt contains the values a user needs to complete device login.
type DevicePrompt struct {
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	UserCode                string `json:"user_code"`
	DeviceCode              string `json:"device_code"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
	Error                   string `json:"error,omitempty"`
	ErrorDescription        string `json:"error_description,omitempty"`
}

// Manager owns OIDC network operations and credential persistence.
type Manager struct {
	TLSVerify        *bool // Explicit login override, persisted only on success.
	configuredServer string
	managedClients   bool
	tlsPolicies      map[string]bool
	tlsMutex         sync.Mutex

	Store         *Store
	HTTPClient    *http.Client // REANA server requests, including discovery.
	IdPHTTPClient *http.Client // Identity-provider requests outside the REANA origin.
	Now           func() time.Time
	Sleep         func(context.Context, time.Duration) error
}

// NewManager creates an authentication manager; transports follow server resolution.
func NewManager() (*Manager, error) {
	store, err := NewStore()
	if err != nil {
		return nil, err
	}
	return &Manager{
		Store:          store,
		managedClients: true,
		Now:            time.Now,
		Sleep: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}, nil
}

// configure resolves the destination before constructing transports. Injected
// clients remain available to library users and tests.
func (m *Manager) configure(serverURL string) error {
	if m.configuredServer == serverURL {
		return nil
	}
	if m.managedClients || m.HTTPClient == nil {
		client, err := m.serverHTTPClient(serverURL)
		if err != nil {
			return err
		}
		m.HTTPClient = client
	}
	if m.managedClients || m.IdPHTTPClient == nil {
		client, err := NewStrictHTTPClient()
		if err != nil {
			return err
		}
		m.IdPHTTPClient = client
	}
	m.configuredServer = serverURL
	return nil
}

func authenticationError(format string, args ...any) error {
	return &AuthenticationError{Message: fmt.Sprintf(format, args...)}
}

func validateOIDCURL(name, value string, required bool) error {
	if value == "" {
		if required {
			return authenticationError(
				"authentication metadata is missing required field: %s",
				name,
			)
		}
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" ||
		parsed.User != nil ||
		parsed.Fragment != "" {
		return authenticationError(
			"authentication metadata field %q must be an HTTPS URL",
			name,
		)
	}
	return nil
}

func validateMetadata(metadata Metadata, deviceRequired bool) error {
	required := map[string]string{
		"issuer":                 metadata.Issuer,
		"authorization_endpoint": metadata.AuthorizationEndpoint,
		"token_endpoint":         metadata.TokenEndpoint,
	}
	for name, value := range required {
		if err := validateOIDCURL(name, value, true); err != nil {
			return err
		}
	}
	if strings.TrimSpace(metadata.CLIClientID) == "" {
		return authenticationError(
			"authentication metadata is missing required field: reana_cli_client_id",
		)
	}
	if err := validateOIDCURL("device_authorization_endpoint", metadata.DeviceAuthorizationEndpoint, deviceRequired); err != nil {
		return err
	}
	return validateOIDCURL(
		"revocation_endpoint",
		metadata.RevocationEndpoint,
		false,
	)
}

func validateStoredMetadata(metadata Metadata) error {
	for name, value := range map[string]string{
		"issuer":         metadata.Issuer,
		"token_endpoint": metadata.TokenEndpoint,
	} {
		if err := validateOIDCURL(name, value, true); err != nil {
			return err
		}
	}
	if strings.TrimSpace(metadata.CLIClientID) == "" {
		return authenticationError(
			"stored authentication metadata is missing the client id; run `reana-client-go login`",
		)
	}
	for name, value := range map[string]string{
		"authorization_endpoint":        metadata.AuthorizationEndpoint,
		"device_authorization_endpoint": metadata.DeviceAuthorizationEndpoint,
		"revocation_endpoint":           metadata.RevocationEndpoint,
	} {
		if err := validateOIDCURL(name, value, false); err != nil {
			return err
		}
	}
	return nil
}

func responseJSON(response *http.Response, target any) error {
	defer response.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return authenticationError(
			"authentication server returned a non-JSON response",
		)
	}
	return nil
}

func rejectRedirect(response *http.Response, description string) error {
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return authenticationError(
			"%s attempted to redirect to %q; refusing to follow an authentication redirect",
			description,
			response.Header.Get("Location"),
		)
	}
	return nil
}

// Discover retrieves and validates OIDC metadata from REANA Server.
func (m *Manager) Discover(
	ctx context.Context,
	serverURL string,
) (Metadata, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return Metadata{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		normalized+discoveryPath,
		nil,
	)
	if err != nil {
		return Metadata{}, err
	}
	if err := m.configure(normalized); err != nil {
		return Metadata{}, err
	}
	warnTLSVerification(m.HTTPClient, normalized)
	response, err := m.HTTPClient.Do(req)
	if err != nil {
		return Metadata{}, ConnectionError(normalized, normalized, err)
	}
	if err := rejectRedirect(response, "authentication metadata discovery"); err != nil {
		response.Body.Close()
		return Metadata{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return Metadata{}, authenticationError(
			"could not discover authentication metadata from %s: HTTP %d",
			normalized,
			response.StatusCode,
		)
	}
	var metadata Metadata
	if err := responseJSON(response, &metadata); err != nil {
		return Metadata{}, err
	}
	if err := validateMetadata(metadata, false); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

type pkcePair struct {
	Verifier  string
	Challenge string
}

func randomURLSafe(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func generatePKCE() (pkcePair, error) {
	verifier, err := randomURLSafe(64)
	if err != nil {
		return pkcePair{}, err
	}
	digest := sha256.Sum256([]byte(verifier))
	return pkcePair{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(digest[:]),
	}, nil
}

// httpClientForURL also applies the server settings to bundled Keycloak.
// Select per request so an external endpoint or another cluster stays strict.
func (m *Manager) httpClientForURL(serverURL, endpoint string) *http.Client {
	if sameHTTPSOrigin(serverURL, endpoint) {
		warnTLSVerification(m.HTTPClient, serverURL)
		return m.HTTPClient
	}
	return m.IdPHTTPClient
}

func (m *Manager) postForm(
	ctx context.Context,
	serverURL, endpoint, description string,
	form url.Values,
	target any,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := m.configure(serverURL); err != nil {
		return nil, err
	}
	response, err := m.httpClientForURL(serverURL, endpoint).Do(req)
	if err != nil {
		return nil, ConnectionError(serverURL, endpoint, err)
	}
	if err := rejectRedirect(response, description); err != nil {
		response.Body.Close()
		return nil, err
	}
	if err := responseJSON(response, target); err != nil {
		return nil, err
	}
	return response, nil
}

func lifetimeExpiry(
	expiresIn *int64,
	field string,
	now time.Time,
) (string, error) {
	if expiresIn == nil {
		return "", nil
	}
	if *expiresIn < 0 || *expiresIn > int64(math.MaxInt64/time.Second) {
		return "", authenticationError(
			"authentication server returned an invalid %s value",
			field,
		)
	}
	return now.UTC().
		Add(time.Duration(*expiresIn) * time.Second).
		Format(time.RFC3339), nil
}

func tokenExpiry(
	accessToken string,
	expiresIn *int64,
	now time.Time,
) (string, error) {
	if expiresIn != nil {
		return lifetimeExpiry(expiresIn, "expires_in", now)
	}
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return "", nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil
	}
	var claims struct {
		ExpiresAt json.Number `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return "", nil
	}
	seconds, err := strconv.ParseInt(string(claims.ExpiresAt), 10, 64)
	if err != nil {
		return "", nil
	}
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339), nil
}

func credentialsFromToken(
	metadata Metadata,
	tokens tokenResponse,
	oldRefreshToken string,
	now time.Time,
) (Credentials, error) {
	if tokens.AccessToken == "" {
		return Credentials{}, authenticationError(
			"authentication server did not return an access token",
		)
	}
	if tokens.RefreshToken == "" {
		tokens.RefreshToken = oldRefreshToken
	}
	accessTokenExpiresAt, err := tokenExpiry(
		tokens.AccessToken,
		tokens.ExpiresIn,
		now,
	)
	if err != nil {
		return Credentials{}, err
	}
	refreshTokenExpiresAt, err := lifetimeExpiry(
		tokens.RefreshExpiresIn,
		"refresh_expires_in",
		now,
	)
	if err != nil {
		return Credentials{}, err
	}
	credentials := Credentials{
		Issuer:                      metadata.Issuer,
		ClientID:                    metadata.CLIClientID,
		TokenEndpoint:               metadata.TokenEndpoint,
		AuthorizationEndpoint:       metadata.AuthorizationEndpoint,
		DeviceAuthorizationEndpoint: metadata.DeviceAuthorizationEndpoint,
		RevocationEndpoint:          metadata.RevocationEndpoint,
		AccessToken:                 tokens.AccessToken,
		AccessTokenExpiresAt:        accessTokenExpiresAt,
		RefreshToken:                tokens.RefreshToken,
		RefreshTokenExpiresAt:       refreshTokenExpiresAt,
	}
	return credentials, nil
}

func refreshTokenRecoveryCredentials(
	metadata Metadata,
	refreshToken string,
) Credentials {
	return Credentials{
		Issuer:                      metadata.Issuer,
		ClientID:                    metadata.CLIClientID,
		TokenEndpoint:               metadata.TokenEndpoint,
		AuthorizationEndpoint:       metadata.AuthorizationEndpoint,
		DeviceAuthorizationEndpoint: metadata.DeviceAuthorizationEndpoint,
		RevocationEndpoint:          metadata.RevocationEndpoint,
		RefreshToken:                refreshToken,
	}
}

func metadataFromCredentials(credentials Credentials) Metadata {
	return Metadata{
		Issuer:                      credentials.Issuer,
		CLIClientID:                 credentials.ClientID,
		TokenEndpoint:               credentials.TokenEndpoint,
		AuthorizationEndpoint:       credentials.AuthorizationEndpoint,
		DeviceAuthorizationEndpoint: credentials.DeviceAuthorizationEndpoint,
		RevocationEndpoint:          credentials.RevocationEndpoint,
	}
}

func (m *Manager) storeTokens(
	serverURL string,
	metadata Metadata,
	tokens tokenResponse,
	oldRefreshToken string,
	makeActive bool,
) (Credentials, error) {
	credentials, err := credentialsFromToken(
		metadata,
		tokens,
		oldRefreshToken,
		m.Now(),
	)
	if err != nil {
		if tokens.RefreshToken != "" && tokens.RefreshToken != oldRefreshToken {
			// The issuer may rotate the refresh token before we reject another
			// response field. Preserve only that replacement, without accepting
			// the invalid access token, so the next command can retry safely.
			_, storeErr := m.Store.Put(
				serverURL,
				refreshTokenRecoveryCredentials(metadata, tokens.RefreshToken),
				false,
			)
			if storeErr != nil {
				return Credentials{}, authenticationError(
					"%v; could not preserve rotated refresh token: %v",
					err,
					storeErr,
				)
			}
		}
		return Credentials{}, err
	}
	if makeActive && m.TLSVerify != nil {
		credentials.TLS = &TLSSettings{Verify: m.TLSVerify}
	}
	return m.Store.Put(serverURL, credentials, makeActive)
}

// LoginDevice performs OIDC device authorization with PKCE.
func (m *Manager) LoginDevice(
	ctx context.Context,
	serverURL string,
	display func(DevicePrompt),
) (Credentials, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return Credentials{}, err
	}
	metadata, err := m.Discover(ctx, normalized)
	if err != nil {
		return Credentials{}, err
	}
	if err := validateMetadata(metadata, true); err != nil {
		return Credentials{}, err
	}
	pkce, err := generatePKCE()
	if err != nil {
		return Credentials{}, err
	}
	var prompt DevicePrompt
	response, err := m.postForm(
		ctx,
		normalized,
		metadata.DeviceAuthorizationEndpoint,
		"device authorization",
		url.Values{
			"client_id":             {metadata.CLIClientID},
			"scope":                 {defaultScopes},
			"code_challenge":        {pkce.Challenge},
			"code_challenge_method": {"S256"},
		},
		&prompt,
	)
	if err != nil {
		return Credentials{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Credentials{}, authenticationError(
			"could not start device login: %s",
			firstNonEmpty(
				prompt.ErrorDescription,
				prompt.Error,
				response.Status,
			),
		)
	}
	if prompt.DeviceCode == "" || prompt.ExpiresIn <= 0 ||
		prompt.ExpiresIn > int64(deviceFlowMax/time.Second) {
		return Credentials{}, authenticationError(
			"device login response did not contain a valid code and expiry",
		)
	}
	promptURL := prompt.VerificationURIComplete
	if promptURL == "" {
		promptURL = prompt.VerificationURI
		if prompt.UserCode == "" {
			return Credentials{}, authenticationError(
				"device login response did not contain a usable verification prompt",
			)
		}
	}
	if err := validateOIDCURL("device verification URI", promptURL, true); err != nil {
		return Credentials{}, authenticationError(
			"device login response did not contain a usable verification prompt",
		)
	}
	if prompt.Interval <= 0 {
		if prompt.Interval < 0 {
			return Credentials{}, authenticationError(
				"device login response contained an invalid polling interval",
			)
		}
		prompt.Interval = 5
	}
	if prompt.Interval > int64(deviceFlowMax/time.Second) {
		return Credentials{}, authenticationError(
			"device login response contained an invalid polling interval",
		)
	}
	display(prompt)
	deadline := m.Now().Add(time.Duration(prompt.ExpiresIn) * time.Second)
	interval := time.Duration(prompt.Interval) * time.Second
	for {
		remaining := deadline.Sub(m.Now())
		if remaining <= 0 {
			return Credentials{}, authenticationError(
				"device login expired; please run login again",
			)
		}
		if interval > remaining {
			interval = remaining
		}
		if err := m.Sleep(ctx, interval); err != nil {
			return Credentials{}, err
		}
		var tokens tokenResponse
		response, err = m.postForm(
			ctx,
			normalized,
			metadata.TokenEndpoint,
			"device token polling",
			url.Values{
				"grant_type": {
					"urn:ietf:params:oauth:grant-type:device_code",
				},
				"device_code":   {prompt.DeviceCode},
				"client_id":     {metadata.CLIClientID},
				"code_verifier": {pkce.Verifier},
			},
			&tokens,
		)
		if err != nil {
			return Credentials{}, err
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return m.storeTokens(normalized, metadata, tokens, "", true)
		}
		switch tokens.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			return Credentials{}, authenticationError(
				"device login expired; please run login again",
			)
		case "access_denied":
			return Credentials{}, authenticationError("device login was denied")
		default:
			return Credentials{}, authenticationError(
				"device login failed: %s",
				firstNonEmpty(
					tokens.ErrorDescription,
					tokens.Error,
					response.Status,
				),
			)
		}
	}
}

type callbackResult struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

// LoginBrowser performs loopback authorization-code login with PKCE.
func (m *Manager) LoginBrowser(
	ctx context.Context,
	serverURL string,
	display func(string),
	open func(string) error,
) (Credentials, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return Credentials{}, err
	}
	metadata, err := m.Discover(ctx, normalized)
	if err != nil {
		return Credentials{}, err
	}
	pkce, err := generatePKCE()
	if err != nil {
		return Credentials{}, err
	}
	state, err := randomURLSafe(32)
	if err != nil {
		return Credentials{}, err
	}
	loopbackPort := 0
	rawLoopbackPort := strings.TrimSpace(os.Getenv(loginLoopbackPortEnv))
	if rawLoopbackPort != "" {
		loopbackPort, err = strconv.Atoi(rawLoopbackPort)
		if err != nil || loopbackPort < 0 || loopbackPort > 65535 {
			return Credentials{}, authenticationError(
				"%s must be an integer between 0 and 65535, got %q",
				loginLoopbackPortEnv,
				rawLoopbackPort,
			)
		}
	}
	listenerAddress := net.JoinHostPort("127.0.0.1", strconv.Itoa(loopbackPort))
	listener, err := net.Listen("tcp", listenerAddress)
	if err != nil {
		if rawLoopbackPort != "" && loopbackPort != 0 {
			return Credentials{}, authenticationError(
				"could not start the login callback listener on %s fixed by %s: %v",
				listenerAddress,
				loginLoopbackPortEnv,
				err,
			)
		}
		return Credentials{}, authenticationError(
			"could not start the login callback listener: %v",
			err,
		)
	}
	defer listener.Close()
	redirectURI := "http://" + listener.Addr().String() + loopbackCallbackURL
	results := make(chan callbackResult, 1)
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	server.Handler = http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != loopbackCallbackURL {
				http.NotFound(writer, request)
				return
			}
			query := request.URL.Query()
			if query.Get("code") == "" && query.Get("error") == "" {
				http.Error(
					writer,
					"Missing authorization response.",
					http.StatusBadRequest,
				)
				return
			}
			result := callbackResult{
				Code:             query.Get("code"),
				State:            query.Get("state"),
				Error:            query.Get("error"),
				ErrorDescription: query.Get("error_description"),
			}
			select {
			case results <- result:
			default:
			}
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			if result.Error != "" {
				_, _ = io.WriteString(
					writer,
					"<html><body><h1>REANA login failed.</h1>"+
						"<p>You can close this tab and return to the terminal.</p></body></html>",
				)
				return
			}
			_, _ = io.WriteString(
				writer,
				"<html><body><h1>REANA login complete.</h1><p>You can close this tab and return to the terminal.</p></body></html>",
			)
		},
	)
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())
	authorizationURL, _ := url.Parse(metadata.AuthorizationEndpoint)
	query := authorizationURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", metadata.CLIClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", defaultScopes)
	query.Set("state", state)
	query.Set("code_challenge", pkce.Challenge)
	query.Set("code_challenge_method", "S256")
	authorizationURL.RawQuery = query.Encode()
	display(authorizationURL.String())
	_ = open(authorizationURL.String())
	var result callbackResult
	select {
	case <-ctx.Done():
		return Credentials{}, authenticationError(
			"browser login timed out; please run login again",
		)
	case result = <-results:
	}
	if result.Error != "" {
		return Credentials{}, authenticationError(
			"browser login failed: %s",
			firstNonEmpty(result.ErrorDescription, result.Error),
		)
	}
	if subtle.ConstantTimeCompare([]byte(result.State), []byte(state)) != 1 {
		return Credentials{}, authenticationError(
			"browser login failed: state parameter mismatch (possible CSRF)",
		)
	}
	if result.Code == "" {
		return Credentials{}, authenticationError(
			"browser login failed: no authorization code was returned",
		)
	}
	var tokens tokenResponse
	response, err := m.postForm(
		ctx,
		normalized,
		metadata.TokenEndpoint,
		"authorization code exchange",
		url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {metadata.CLIClientID},
			"code":          {result.Code},
			"redirect_uri":  {redirectURI},
			"code_verifier": {pkce.Verifier},
		},
		&tokens,
	)
	if err != nil {
		return Credentials{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Credentials{}, authenticationError(
			"browser login failed: %s",
			firstNonEmpty(
				tokens.ErrorDescription,
				tokens.Error,
				response.Status,
			),
		)
	}
	return m.storeTokens(normalized, metadata, tokens, "", true)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown error"
}

func accessTokenValid(credentials Credentials, now time.Time) bool {
	if credentials.AccessToken == "" {
		return false
	}
	// An empty or otherwise unparseable expiry must fail closed (trigger a
	// refresh) rather than being treated as valid forever.
	expiresAt, err := time.Parse(time.RFC3339, credentials.AccessTokenExpiresAt)
	return err == nil && expiresAt.Add(-expiryLeeway).After(now)
}

// AccessToken returns a valid stored token, refreshing it when necessary.
func (m *Manager) AccessToken(
	ctx context.Context,
	serverURL string,
) (string, error) {
	if serverURL == "" {
		var err error
		serverURL, err = m.Store.ActiveServer()
		if err != nil {
			return "", err
		}
	}
	if serverURL == "" {
		return "", authenticationError(
			NoServerMessage,
		)
	}
	credentials, err := m.Store.Get(serverURL)
	if err != nil {
		return "", err
	}
	if accessTokenValid(credentials, m.Now()) {
		return credentials.AccessToken, nil
	}
	refreshed, err := m.Refresh(ctx, serverURL, credentials)
	if err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

// Refresh rotates an expired access token with cross-process serialization.
func (m *Manager) Refresh(
	ctx context.Context,
	serverURL string,
	credentials Credentials,
) (Credentials, error) {
	normalized, err := NormalizeServerURL(serverURL)
	if err != nil {
		return Credentials{}, err
	}
	deadline := time.Now().Add(refreshLockWait)
	var refreshLock *os.File
	for refreshLock == nil {
		refreshLock, err = m.Store.tryRefreshLock(normalized)
		if err != nil {
			return Credentials{}, err
		}
		if refreshLock != nil {
			break
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return Credentials{}, authenticationError(
				"timed out waiting for another credential refresh; please retry",
			)
		}
		finished, waitErr := m.Store.waitRefreshLock(
			normalized,
			remaining,
		)
		if waitErr != nil {
			return Credentials{}, waitErr
		}
		if !finished {
			return Credentials{}, authenticationError(
				"timed out waiting for another credential refresh; please retry",
			)
		}
		newCredentials, getErr := m.Store.Get(normalized)
		if getErr != nil {
			return Credentials{}, getErr
		}
		if accessTokenValid(newCredentials, m.Now()) {
			return newCredentials, nil
		}
	}
	defer releaseLock(refreshLock)
	credentials, err = m.Store.Get(normalized)
	if err != nil {
		return Credentials{}, err
	}
	if credentials.RefreshToken == "" {
		return Credentials{}, authenticationError(
			"No usable credentials for %s. Run `reana-client-go login --server %s`.",
			ServerDescription(normalized),
			normalized,
		)
	}
	metadata := metadataFromCredentials(credentials)
	if err := validateStoredMetadata(metadata); err != nil {
		return Credentials{}, err
	}
	startedEpoch := credentials.CredentialEpoch
	refreshToken := credentials.RefreshToken
	var tokens tokenResponse
	response, err := m.postForm(
		ctx,
		normalized,
		metadata.TokenEndpoint,
		"token refresh",
		url.Values{
			"grant_type":    {"refresh_token"},
			"client_id":     {credentials.ClientID},
			"refresh_token": {refreshToken},
		},
		&tokens,
	)
	if err != nil {
		return Credentials{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if tokens.Error == "invalid_grant" {
			cleared, clearErr := m.Store.ClearTokens(normalized, refreshToken)
			if clearErr != nil {
				return Credentials{}, clearErr
			}
			if cleared {
				return Credentials{}, authenticationError(
					"No usable credentials for %s. Run `reana-client-go login --server %s`.",
					ServerDescription(normalized),
					normalized,
				)
			}
			return Credentials{}, authenticationError(
				"credentials were changed by another process; please retry",
			)
		}
		return Credentials{}, authenticationError(
			"could not refresh authentication credentials: %s",
			firstNonEmpty(
				tokens.ErrorDescription,
				tokens.Error,
				response.Status,
			),
		)
	}
	updated, err := credentialsFromToken(
		metadata,
		tokens,
		refreshToken,
		m.Now(),
	)
	if err != nil {
		if tokens.RefreshToken != "" && tokens.RefreshToken != refreshToken {
			recovery := refreshTokenRecoveryCredentials(
				metadata,
				tokens.RefreshToken,
			)
			_, matched, storeErr := m.Store.PutIfEpoch(
				normalized,
				recovery,
				startedEpoch,
			)
			if storeErr != nil {
				return Credentials{}, authenticationError(
					"%v; could not preserve rotated refresh token: %v",
					err,
					storeErr,
				)
			}
			if !matched {
				revokeContext, cancel := context.WithTimeout(
					context.Background(),
					30*time.Second,
				)
				defer cancel()
				m.revokeBestEffort(
					revokeContext,
					normalized,
					metadata,
					tokens.RefreshToken,
				)
			}
		}
		return Credentials{}, err
	}
	stored, matched, err := m.Store.PutIfEpoch(
		normalized,
		updated,
		startedEpoch,
	)
	if err != nil {
		return Credentials{}, err
	}
	if !matched {
		revokeContext, cancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
		defer cancel()
		m.revokeBestEffort(
			revokeContext,
			normalized,
			metadata,
			updated.RefreshToken,
		)
		return Credentials{}, authenticationError(
			"No usable credentials for %s. Run `reana-client-go login --server %s`.",
			ServerDescription(normalized),
			normalized,
		)
	}
	return stored, nil
}

func (m *Manager) revokeBestEffort(
	ctx context.Context,
	serverURL string,
	metadata Metadata,
	refreshToken string,
) string {
	if metadata.RevocationEndpoint == "" || refreshToken == "" {
		return ""
	}
	if err := m.configure(serverURL); err != nil {
		return err.Error()
	}
	return revokeWithClient(
		ctx,
		m.httpClientForURL(serverURL, metadata.RevocationEndpoint),
		serverURL,
		metadata,
		refreshToken,
	)
}

// revokeWithClient uses an already resolved transport and never reads the store.
func revokeWithClient(
	ctx context.Context,
	client *http.Client,
	serverURL string,
	metadata Metadata,
	refreshToken string,
) string {
	form := url.Values{
		"client_id":       {metadata.CLIClientID},
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		metadata.RevocationEndpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Sprintf(
			"Could not prepare token revocation for %s. Check the saved revocation endpoint.",
			ServerDescription(serverURL),
		)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return ConnectionError(
			serverURL,
			metadata.RevocationEndpoint,
			err,
		).Error()
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		// A redirect Location can contain credentials; report only its status.
		return fmt.Sprintf(
			"Remote token revocation for %s failed with HTTP %d. Refusing to follow a redirect on an authentication request.",
			ServerDescription(serverURL),
			response.StatusCode,
		)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Sprintf(
			"Remote token revocation for %s failed with HTTP %d.",
			ServerDescription(serverURL), response.StatusCode,
		)
	}
	return ""
}

// Logout revokes the current refresh token when possible and clears local tokens.
func (m *Manager) Logout(
	ctx context.Context,
	serverURL string,
) (string, error) {
	if serverURL == "" {
		var err error
		serverURL, err = m.Store.ActiveServer()
		if err != nil {
			return "", err
		}
	}
	if serverURL == "" {
		return "", authenticationError(
			NoServerMessage,
		)
	}
	if err := m.configure(serverURL); err != nil {
		return "", err
	}
	return m.Store.Logout(serverURL, func(credentials Credentials) string {
		return m.revokeBestEffort(
			ctx,
			serverURL,
			metadataFromCredentials(credentials),
			credentials.RefreshToken,
		)
	})
}

// IsJWT reports whether a token has the JWT compact serialization shape.
func IsJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}

// IsAuthenticationError reports whether err is intended as a user-facing auth error.
func IsAuthenticationError(err error) bool {
	var target *AuthenticationError
	return errors.As(err, &target)
}
