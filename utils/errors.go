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
	"net/url"
	"os"
	"reflect"
)

// HandleApiError Handles API Error response which contains a payload with a message
// Returns the original error when this doesn't happen
func HandleApiError(err error) error {
	_, isUrlErr := err.(*url.Error)
	if isUrlErr {
		return fmt.Errorf(
			"'%s' not found, please verify the provided server URL or check your internet connection",
			os.Getenv("REANA_SERVER_URL"),
		)
	}

	if errValue := reflect.Indirect(reflect.ValueOf(err)); errValue.Kind() == reflect.Struct {
		if payload := reflect.Indirect(errValue.FieldByName("Payload")); payload.Kind() == reflect.Struct {
			if message := payload.FieldByName("Message"); message.Kind() == reflect.String {
				return errors.New(message.String())
			}
		}
	}

	return err
}
