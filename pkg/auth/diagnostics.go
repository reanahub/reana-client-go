/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// NoServerMessage explains first-use configuration without a localhost default.
const NoServerMessage = "No REANA server is configured. Run `reana-client-go login --server https://your-reana-server` using the URL from your REANA administrator."

// ServerDescription identifies the destination and selection source.
func ServerDescription(serverURL string) string {
	source := "saved login"
	return fmt.Sprintf("%s (from %s)", serverURL, source)
}

// ConnectionError classifies transport errors without printing URLs, queries or tokens
// from the wrapped error. The cause remains reachable through errors.As.
func ConnectionError(serverURL, endpoint string, err error) error {
	reason := "The network request failed. Check the server address and network connection."
	var authority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	var verification *tls.CertificateVerificationError
	var dns *net.DNSError
	var network net.Error
	switch {
	case errors.As(err, &hostname):
		reason = "The TLS certificate does not match the hostname. Check the server URL and contact the administrator."
	case errors.As(err, &invalid) && invalid.Reason == x509.Expired:
		reason = "The TLS certificate has expired or is not yet valid. Check the clock and ask the administrator to renew it."
	case errors.As(err, &authority):
		reason = "The TLS certificate is not trusted. "
		if os.Getenv(caCertsEnv) != "" {
			reason += "The CA bundle in REANA_SERVER_CA_CERTS does not trust this certificate."
		} else {
			reason += "Ask the administrator for a CA bundle and set REANA_SERVER_CA_CERTS."
			if sameHTTPSOrigin(serverURL, endpoint) {
				reason += fmt.Sprintf(" For a trusted development server, run `reana-client-go login --server %s --no-tls-verify` to disable verification.", serverURL)
			} else {
				reason += " The REANA server's TLS bypass does not apply to this identity provider."
			}
		}
	case errors.As(err, &verification):
		reason = "TLS negotiation or certificate verification failed. Check certificate trust, validity and hostname."
	case errors.As(err, &dns):
		reason = "The hostname could not be resolved. Check the server URL and DNS."
	case errors.Is(err, syscall.ECONNREFUSED):
		reason = "The connection was refused. Check that the server is running and reachable."
	case errors.As(err, &network) && network.Timeout():
		reason = "The connection timed out. Check connectivity and try again."
	}
	// Raw request errors can include secrets in queries. Log types, retaining the
	// typed cause for inspection without exposing request data at DEBUG.
	log.Debugf("Connection failure type: %T", err)
	return &AuthenticationError{
		Message: fmt.Sprintf(
			"Could not connect to %s: %s",
			ServerDescription(serverURL),
			reason,
		),
		Cause: err,
	}
}

// ForServer adds invocation context while retaining the original error chain.
func ForServer(serverURL, source string, err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if !strings.Contains(message, serverURL) {
		message = fmt.Sprintf(
			"REANA server: %s (from %s)\n%s",
			serverURL,
			source,
			message,
		)
	} else if source != "saved login" {
		message = strings.ReplaceAll(message, "(from saved login)", "(from "+source+")")
	}
	return &AuthenticationError{Message: message, Cause: err}
}
