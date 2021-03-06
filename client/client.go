package client

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"os"

	"github.com/go-openapi/strfmt"

	httptransport "github.com/go-openapi/runtime/client"
)

var apiClient *API

func ApiClient() (*API, error) {
	if apiClient == nil {
		var err error
		apiClient, err = newApiClient()
		if err != nil {
			return nil, err
		}
	}
	return apiClient, nil
}

func newApiClient() (*API, error) {
	// disable certificate security checks
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	// parse REANA server URL
	serverURL := os.Getenv("REANA_SERVER_URL")
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, errors.New("environment variable REANA_SERVER_URL is not set")
	}

	// create the transport
	transport := httptransport.New(u.Host, "", []string{"https"})

	// create the API client, with the transport
	return New(transport, strfmt.Default), nil
}
