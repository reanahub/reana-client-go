// This file is part of REANA.
// Copyright (C) 2026 CERN.

package client

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/reanahub/reana-client-go/client/operations"

	"github.com/spf13/viper"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return function(request)
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (body *trackedBody) Close() error {
	body.closed = true
	return nil
}

func TestBoundedResponseBodyRejectsBytesPastLimit(t *testing.T) {
	body := &boundedResponseBody{
		body:      io.NopCloser(strings.NewReader("four")),
		remaining: 3,
	}
	contents, err := io.ReadAll(body)
	if err == nil ||
		!strings.Contains(err.Error(), ErrResponseTooLarge.Error()) {
		t.Fatalf("expected response limit error, got %q and %v", contents, err)
	}
	if !bytes.Equal(contents, []byte("fou")) {
		t.Fatalf("unexpected bounded contents %q", contents)
	}
}

func TestBoundedResponseBodyAcceptsExactLimit(t *testing.T) {
	body := &boundedResponseBody{
		body:      io.NopCloser(strings.NewReader("three")),
		remaining: 5,
	}
	contents, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, []byte("three")) {
		t.Fatalf("unexpected contents %q", contents)
	}
}

func TestBoundedResponseBodyCloseDelegatesToUnderlyingBody(t *testing.T) {
	underlying := &trackedBody{Reader: strings.NewReader("response")}
	body := &boundedResponseBody{body: underlying, remaining: 8}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if !underlying.closed {
		t.Fatal("expected the underlying body to be closed")
	}
}

func TestKnownLengthMultipartTransportSpoolsBodyForExactContentLength(
	t *testing.T,
) {
	const payload = "--boundary\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\ndata\r\n--boundary--\r\n"
	original := &trackedBody{Reader: strings.NewReader(payload)}
	request := httptest.NewRequest(
		http.MethodPost,
		"https://reana.test/upload",
		original,
	)
	request.ContentLength = -1
	request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	request.Header.Set("Transfer-Encoding", "chunked")

	var seen *http.Request
	var spooled []byte
	var readErr error
	transport := &knownLengthMultipartTransport{
		transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			// The spooled temp file is closed by a defer once RoundTrip
			// returns, so it must be read from inside this callback.
			spooled, readErr = io.ReadAll(r.Body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		}),
	}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if !original.closed {
		t.Fatal("expected the original body to be closed after spooling")
	}
	if seen.ContentLength != int64(len(payload)) {
		t.Fatalf(
			"Content-Length = %d, want %d",
			seen.ContentLength,
			len(payload),
		)
	}
	if seen.TransferEncoding != nil {
		t.Fatalf("TransferEncoding = %v, want nil", seen.TransferEncoding)
	}
	if seen.Header.Get("Transfer-Encoding") != "" {
		t.Fatal("Transfer-Encoding header was not removed")
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(spooled) != payload {
		t.Fatalf("spooled body = %q, want %q", spooled, payload)
	}
}

func TestKnownLengthMultipartTransportPassesThroughNonMultipartRequests(
	t *testing.T,
) {
	original := &trackedBody{Reader: strings.NewReader(`{"a":1}`)}
	request := httptest.NewRequest(
		http.MethodPost,
		"https://reana.test/api",
		original,
	)
	request.ContentLength = -1
	request.Header.Set("Content-Type", "application/json")

	var seen *http.Request
	transport := &knownLengthMultipartTransport{
		transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		}),
	}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if seen != request {
		t.Fatal("expected the original request to pass through unchanged")
	}
	if original.closed {
		t.Fatal(
			"did not expect the body to be closed for a passthrough request",
		)
	}
}

func TestKnownLengthMultipartTransportPassesThroughWhenContentLengthKnown(
	t *testing.T,
) {
	original := &trackedBody{Reader: strings.NewReader("data")}
	request := httptest.NewRequest(
		http.MethodPost,
		"https://reana.test/upload",
		original,
	)
	request.ContentLength = 4
	request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	transport := &knownLengthMultipartTransport{
		transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		}),
	}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if original.closed {
		t.Fatal(
			"did not expect the body to be spooled when Content-Length is already known",
		)
	}
}

func TestKnownLengthMultipartTransportSetsNoBodyForEmptyMultipartPayload(
	t *testing.T,
) {
	original := &trackedBody{Reader: strings.NewReader("")}
	request := httptest.NewRequest(
		http.MethodPost,
		"https://reana.test/upload",
		original,
	)
	request.ContentLength = -1
	request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	var seen *http.Request
	transport := &knownLengthMultipartTransport{
		transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		}),
	}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if seen.Body != http.NoBody {
		t.Fatalf("body = %v, want http.NoBody", seen.Body)
	}
	if seen.ContentLength != 0 {
		t.Fatalf("Content-Length = %d, want 0", seen.ContentLength)
	}
}

func TestBoundedResponseTransportPropagatesUnderlyingTransportError(
	t *testing.T,
) {
	transport := &boundedResponseTransport{
		transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection reset")
		}),
	}
	_, err := transport.RoundTrip(
		httptest.NewRequest(http.MethodGet, "https://reana.test", nil),
	)
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("expected the underlying transport error, got %v", err)
	}
}

func TestBoundedResponseTransportRejectsDeclaredOversize(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader("response")}
	transport := &boundedResponseTransport{transport: roundTripFunc(func(
		*http.Request,
	) (*http.Response, error) {
		return &http.Response{
			Body:          body,
			ContentLength: maxAPIResponseBytes + 1,
		}, nil
	})}
	_, err := transport.RoundTrip(
		httptest.NewRequest(http.MethodGet, "https://reana.test", nil),
	)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected %v, got %v", ErrResponseTooLarge, err)
	}
	if !body.closed {
		t.Error("expected rejected response body to be closed")
	}
}

func TestAPIClientBoundsControlResponses(t *testing.T) {
	payload := `{"status":"` + strings.Repeat("x", maxAPIResponseBytes) + `"}`
	server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = w.Write([]byte(payload))
		}),
	)
	defer server.Close()
	viper.Set("server-url", server.URL)
	t.Cleanup(viper.Reset)
	t.Setenv("REANA_SERVER_TLS_VERIFY", "false")

	api, err := ControlAPIClient("token")
	if err != nil {
		t.Fatal(err)
	}
	params := operations.NewGetWorkflowStatusParams().
		WithWorkflowIDOrName("analysis")
	_, err = api.Operations.GetWorkflowStatus(params, nil)
	if err == nil ||
		!strings.Contains(err.Error(), ErrResponseTooLarge.Error()) {
		t.Fatalf("expected bounded response error, got %v", err)
	}
}
