package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/go-openapi/strfmt"

	httptransport "github.com/go-openapi/runtime/client"
)

var ApiClient = newApiClient()

func newApiClient() *API {
	// disable certificate security checks
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	// parse REANA server URL
	serverURL := os.Getenv("REANA_SERVER_URL")
	u, err := url.Parse(serverURL)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	if u.Host == "" {
		fmt.Println("Error: Environment variable REANA_SERVER_URL is not set")
		os.Exit(1)
	}

	// create the transport
	transport := httptransport.New(u.Host, "", []string{"https"})

	// create the API client, with the transport
	return New(transport, strfmt.Default)
}
