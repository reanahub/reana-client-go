// Package client provides the automatically generated API client, provided by the swagger tool.
package client

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"

	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	httptransport "github.com/go-openapi/runtime/client"
)

// ApiClient provides a new API client used to communicate with the REANA server.
func ApiClient() (*API, error) {
	// disable certificate security checks
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	// parse REANA server URL
	serverURL := viper.GetString("server-url")
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, errors.New("environment variable REANA_SERVER_URL is not set")
	}

	// create the transport
	transport := httptransport.New(u.Host, "", []string{"https"})
	transport.SetLogger(log.StandardLogger())
	transport.SetDebug(log.GetLevel() == log.DebugLevel)

	// create the API client, with the transport
	return New(transport, strfmt.Default), nil
}
