/*
This file is part of REANA.
Copyright (C) 2022, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package errorhandler

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"

	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/viper"
)

type testApiError struct {
	Payload struct{ Message string }
}

func (e *testApiError) Error() string { return e.Payload.Message }

type testPointerApiError struct {
	Payload *struct{ Message *string }
}

func (e *testPointerApiError) Error() string { return *e.Payload.Message }

func TestHandleApiError(t *testing.T) {
	serverURL := "https://localhost:8080"
	viper.Set("server-url", serverURL)
	t.Cleanup(func() {
		viper.Reset()
	})

	apiError := testApiError{
		Payload: struct{ Message string }{Message: "API Error"},
	}
	pointerMessage := "Pointer API Error"
	pointerApiError := testPointerApiError{
		Payload: &struct{ Message *string }{Message: &pointerMessage},
	}
	otherError := errors.New("other Error")

	tests := map[string]struct {
		arg  error
		want string
	}{
		"api error": {
			arg:  &apiError,
			want: apiError.Error(),
		},
		"wrapped api error": {
			arg:  fmt.Errorf("cannot complete request: %w", &apiError),
			want: apiError.Error(),
		},
		"generated pointer api error": {
			arg:  &pointerApiError,
			want: pointerApiError.Error(),
		},
		"other error": {
			arg:  otherError,
			want: otherError.Error(),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := HandleApiError(test.arg)
			if got.Error() != test.want {
				t.Errorf("Expected %s, got %s", test.want, got)
			}
		})
	}
}

func TestTransportDiagnostics(t *testing.T) {
	server := "https://reana.example.org"
	viper.Set("server-url", server)
	t.Cleanup(viper.Reset)
	t.Setenv("REANA_SERVER_CA_CERTS", "")
	for _, test := range []struct {
		name     string
		cause    error
		endpoint string
		want     string
		bypass   bool
	}{
		{"certificate", x509.UnknownAuthorityError{}, server, "certificate is not trusted", true},
		{"external issuer", x509.UnknownAuthorityError{}, "https://iam.example.org", "does not apply to this identity provider", false},
		{"dns", &net.DNSError{Name: "secret-request-data", Err: "not found"}, server, "hostname could not be resolved", false},
		{"refused", syscall.ECONNREFUSED, server, "connection was refused", false},
		{"timeout", os.ErrDeadlineExceeded, server, "connection timed out", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			transport := &url.Error{
				Op:  "Get",
				URL: test.endpoint + "/api?secret=query",
				Err: test.cause,
			}
			got := HandleApiError(fmt.Errorf("request failed: %w", transport))
			if !strings.Contains(got.Error(), server+" (from saved login)") ||
				!strings.Contains(got.Error(), test.want) {
				t.Fatalf("missing diagnostic: %v", got)
			}
			if strings.Contains(got.Error(), "--no-tls-verify") != test.bypass {
				t.Fatalf("incorrect bypass advice: %v", got)
			}
			if strings.Contains(got.Error(), "secret") ||
				!errors.Is(got, test.cause) {
				t.Fatalf("cause lost or request data exposed: %v", got)
			}
			if HandleApiError(got) != got {
				t.Fatal("already classified error was replaced")
			}
		})
	}
	// An explicit operation target must survive the global API handler.
	other := "https://other.example.org"
	classified := auth.ConnectionError(other, other, syscall.ECONNREFUSED)
	if HandleApiError(classified) != classified {
		t.Fatal("operation target was replaced by the selected server")
	}
}
