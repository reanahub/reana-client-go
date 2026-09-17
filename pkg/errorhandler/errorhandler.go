/*
This file is part of REANA.
Copyright (C) 2022, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package errorhandler gives utility functions to handle errors.
package errorhandler

import (
	"errors"
	"net/url"
	"reflect"

	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/viper"
)

// HandleApiError Handles API Error response which contains a payload with a message
// Returns the original error when this doesn't happen
func HandleApiError(err error) error {
	if errors.Is(err, client.ErrResponseTooLarge) {
		return client.ErrResponseTooLarge
	}
	var authenticationError *auth.AuthenticationError
	if errors.As(err, &authenticationError) {
		return err
	}
	var transportError *url.Error
	if errors.As(err, &transportError) {
		return auth.ConnectionError(
			viper.GetString("server-url"), transportError.URL, err,
		)
	}

	for current := err; current != nil; current = errors.Unwrap(current) {
		errValue := reflect.Indirect(reflect.ValueOf(current))
		if errValue.Kind() == reflect.Struct {
			payload := reflect.Indirect(errValue.FieldByName("Payload"))
			if payload.Kind() == reflect.Struct {
				message := reflect.Indirect(payload.FieldByName("Message"))
				if message.Kind() == reflect.String {
					return errors.New(message.String())
				}
			}
		}
	}

	return err
}
