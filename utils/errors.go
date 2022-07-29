/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
)

var UnexpectedErrorMsg = "unexpected error"

// HandleBasicApiError Handles API Error response which contains a payload with a message
// Returns an unexpected error when this doesn't happen
func HandleBasicApiError(err error) error {
	urlErr, isUrlErr := err.(*url.Error)
	if isUrlErr {
		return fmt.Errorf(
			"'%s' not found, please verify the provided server URL or check your internet connection",
			urlErr.Err.(*net.OpError).Err.(*net.DNSError).Name,
		)
	}

	apiErr := reflect.ValueOf(err).Elem()
	apiErrType := apiErr.Type()
	if apiErr.NumField() > 0 && apiErrType.Field(0).Name == "Payload" {
		payload := apiErr.Field(0).Elem()
		payloadType := payload.Type()
		if payload.NumField() > 0 && payloadType.Field(0).Name == "Message" {
			msg := payload.Field(0).String()
			return errors.New(msg)
		}
	}

	return errors.New(UnexpectedErrorMsg)
}
