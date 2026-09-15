// Package client provides the automatically generated API client, provided by the swagger tool.
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/auth"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	httptransport "github.com/go-openapi/runtime/client"
)

const (
	apiRequestTimeout   = 5 * time.Minute
	maxAPIResponseBytes = 16 * 1024 * 1024
)

// ErrResponseTooLarge identifies a bounded control-plane response overflow.
var ErrResponseTooLarge = errors.New(
	"REANA server response exceeds 16 MiB limit",
)

type boundedResponseBody struct {
	body      io.ReadCloser
	remaining int64
}

func (body *boundedResponseBody) Read(buffer []byte) (int, error) {
	if body.remaining == 0 {
		var probe [1]byte
		read, err := body.body.Read(probe[:])
		if read > 0 {
			return 0, ErrResponseTooLarge
		}
		return 0, err
	}
	if int64(len(buffer)) > body.remaining {
		buffer = buffer[:body.remaining]
	}
	read, err := body.body.Read(buffer)
	body.remaining -= int64(read)
	return read, err
}

func (body *boundedResponseBody) Close() error {
	return body.body.Close()
}

type boundedResponseTransport struct {
	transport http.RoundTripper
}

// knownLengthMultipartTransport gives generated multipart requests an exact
// Content-Length. go-openapi streams multipart bodies through an io.Pipe, which
// otherwise makes Go send them chunked; deployed uWSGI post-buffering does not
// deliver such request bodies to the application.
type knownLengthMultipartTransport struct {
	transport http.RoundTripper
}

func (transport *knownLengthMultipartTransport) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	mediaType, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if request.Body == nil || request.Body == http.NoBody ||
		request.ContentLength > 0 || mediaType != "multipart/form-data" {
		return transport.transport.RoundTrip(request)
	}

	body, err := os.CreateTemp("", "reana-multipart-*")
	if err != nil {
		return nil, fmt.Errorf("could not spool multipart request: %w", err)
	}
	path := body.Name()
	defer func() {
		_ = body.Close()
		_ = os.Remove(path)
	}()

	length, copyErr := io.Copy(body, request.Body)
	closeErr := request.Body.Close()
	if copyErr != nil {
		return nil, fmt.Errorf("could not spool multipart request: %w", copyErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf(
			"could not close multipart request: %w",
			closeErr,
		)
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("could not rewind multipart request: %w", err)
	}

	request.Body = body
	request.ContentLength = length
	request.TransferEncoding = nil
	request.Header.Del("Transfer-Encoding")
	if length == 0 {
		request.Body = http.NoBody
	}
	return transport.transport.RoundTrip(request)
}

func (transport *boundedResponseTransport) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	response, err := transport.transport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response.ContentLength > maxAPIResponseBytes {
		_ = response.Body.Close()
		return nil, ErrResponseTooLarge
	}
	response.Body = &boundedResponseBody{
		body:      response.Body,
		remaining: maxAPIResponseBytes,
	}
	return response, nil
}

// AuthenticatedClient combines the generated API with authentication helpers.
type AuthenticatedClient struct {
	*API

	httpClient *http.Client
}

func validateBearerTransportURL(u *url.URL) error {
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if strings.EqualFold(host, "localhost") ||
			(ip != nil && ip.IsLoopback()) {
			return nil
		}
		return fmt.Errorf(
			"refusing to send a bearer token over cleartext HTTP to %q; use HTTPS",
			u.Host,
		)
	}
	return fmt.Errorf("unsupported REANA server URL scheme %q", u.Scheme)
}

// ApiClient provides an uncapped API client used for ordinary REANA operations.
func ApiClient(tokens ...string) (*AuthenticatedClient, error) {
	return newAPIClient(tokenOverride(tokens), 0, false)
}

// ControlAPIClient provides a bounded client for small control-plane responses.
func ControlAPIClient(tokens ...string) (*AuthenticatedClient, error) {
	return newAPIClient(tokenOverride(tokens), apiRequestTimeout, true)
}

func tokenOverride(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}

// AccessToken returns an explicit token override or a renewable stored OIDC
// access token for the configured REANA server.
func AccessToken(ctx context.Context, token string) (string, error) {
	if token != "" {
		return token, nil
	}
	manager, err := auth.NewManager()
	if err != nil {
		return "", err
	}
	return manager.AccessToken(ctx, viper.GetString("server-url"))
}

// StreamingHTTPClient returns a client for large raw request bodies and bounded
// control-plane responses. It has no whole-request deadline, but bounds the
// wait for response headers after the body has been transmitted.
func StreamingHTTPClient() (*http.Client, *url.URL, error) {
	normalized, err := auth.NormalizeServerURL(viper.GetString("server-url"))
	if err != nil {
		return nil, nil, err
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return nil, nil, err
	}
	if err := validateBearerTransportURL(u); err != nil {
		return nil, nil, err
	}
	httpClient, err := auth.NewHTTPClient()
	if err != nil {
		return nil, nil, err
	}
	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		return nil, nil, errors.New("unexpected REANA HTTP transport")
	}
	transport.ResponseHeaderTimeout = apiRequestTimeout
	httpClient.Timeout = 0
	httpClient.Transport = &boundedResponseTransport{transport: transport}
	return httpClient, u, nil
}

func newAPIClient(
	token string,
	requestTimeout time.Duration,
	boundResponses bool,
) (*AuthenticatedClient, error) {
	// parse REANA server URL
	normalized, err := auth.NormalizeServerURL(viper.GetString("server-url"))
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return nil, err
	}
	if err := validateBearerTransportURL(u); err != nil {
		return nil, err
	}

	httpClient, err := auth.NewHTTPClient()
	if err != nil {
		return nil, err
	}
	tokenProvider := func(ctx context.Context) (string, error) {
		return AccessToken(ctx, token)
	}

	var transport http.RoundTripper = httpClient.Transport
	if boundResponses {
		transport = &boundedResponseTransport{
			transport: &knownLengthMultipartTransport{transport: transport},
		}
	}
	httpClient.Timeout = requestTimeout
	httpClient.Transport = transport
	apiTransport := httptransport.NewWithClient(
		u.Host,
		strings.TrimRight(u.EscapedPath(), "/"),
		[]string{u.Scheme},
		httpClient,
	)
	apiTransport.SetLogger(log.StandardLogger())
	// Request dumps include Authorization headers and request bodies.
	apiTransport.SetDebug(false)
	apiTransport.DefaultAuthentication = runtime.ClientAuthInfoWriterFunc(
		func(request runtime.ClientRequest, registry strfmt.Registry) error {
			accessToken, err := tokenProvider(context.Background())
			if err != nil {
				return err
			}
			return request.SetHeaderParam(
				"Authorization",
				"Bearer "+accessToken,
			)
		},
	)
	apiTransport.Consumers["application/zip"] = runtime.ByteStreamConsumer()

	log.Info("Connecting to ", normalized)

	return &AuthenticatedClient{
		API:        New(apiTransport, strfmt.Default),
		httpClient: httpClient,
	}, nil
}

// GetInteractiveSessionSecret fetches the short-lived secret used in an
// interactive session URL.
func (c *AuthenticatedClient) GetInteractiveSessionSecret(
	ctx context.Context,
	workflow string,
) (string, error) {
	params := operations.NewGetInteractiveSessionSecretParams().
		WithContext(ctx).
		WithWorkflowIDOrName(workflow)
	response, err := c.Operations.GetInteractiveSessionSecret(params, nil)
	if err != nil {
		return "", err
	}
	if response.Payload == nil || response.Payload.SessionSecret == nil ||
		*response.Payload.SessionSecret == "" {
		return "", errors.New("interactive session secret is empty")
	}
	return *response.Payload.SessionSecret, nil
}
