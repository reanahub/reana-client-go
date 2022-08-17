/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

/*
Package validator provides functions that validate given configurations or command flags.

In case of a failed validation, every function in this package returns an error explaining why it failed.
Otherwise, they return nil, meaning that the validation was successful.
*/
package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"golang.org/x/exp/slices"
)

const (
	InvalidAccessTokenMsg = "please provide your access token by using the -t/--access-token flag, or by setting the REANA_ACCESS_TOKEN environment variable"
	InvalidServerURLMsg   = "please set REANA_SERVER_URL environment variable"
	InvalidWorkflowMsg    = "workflow name must be provided either with `--workflow` option or with REANA_WORKON environment variable"
)

// ValidateAccessToken verifies if the access token has been set, ignoring any white spaces.
func ValidateAccessToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New(InvalidAccessTokenMsg)
	}
	return nil
}

// ValidateServerURL verifies if REANA's server URL has been set, ignoring any white spaces.
func ValidateServerURL(serverURL string) error {
	if strings.TrimSpace(serverURL) == "" {
		return errors.New(InvalidServerURLMsg)
	}
	return nil
}

// ValidateWorkflow verifies if the workflow's name has been set, ignoring any white spaces.
func ValidateWorkflow(workflow string) error {
	if strings.TrimSpace(workflow) == "" {
		return errors.New(InvalidWorkflowMsg)
	}
	return nil
}

// ValidateChoice verifies if the given argument (arg) is part of the slice of available choices.
// The third parameter, name, is the name of the argument/flag that should be displayed if the validation fails.
func ValidateChoice(arg string, choices []string, name string) error {
	if !slices.Contains(choices, arg) {
		return fmt.Errorf(
			"invalid value for '%s': '%s' is not part of '%s'",
			name,
			arg,
			strings.Join(choices, "', '"),
		)
	}
	return nil
}

func ValidateAtLeastOne(f *pflag.FlagSet, options []string) error {
	for _, option := range options {
		if f.Changed(option) {
			return nil
		}
	}
	return fmt.Errorf(
		"at least one of the options: '%s' is required",
		strings.Join(options, "', '"),
	)
}
