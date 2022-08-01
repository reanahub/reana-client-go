package client

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"

	"github.com/go-openapi/strfmt"

	. "reanahub/reana-client-go/config"

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
	u, err := url.Parse(Config.ServerURL)
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
