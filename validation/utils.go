/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package validation

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

func ValidateAccessToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New(
			"please provide your access token by using the -t/--access-token flag, or by setting the REANA_ACCESS_TOKEN environment variable",
		)
	}
	return nil
}

func ValidateServerURL(serverURL string) error {
	if strings.TrimSpace(serverURL) == "" {
		return errors.New("please set REANA_SERVER_URL environment variable")
	}
	return nil
}

func ValidateWorkflow(workflow string) error {
	if strings.TrimSpace(workflow) == "" {
		return errors.New(
			"workflow name must be provided either with `--workflow` option or with REANA_WORKON environment variable",
		)
	}
	return nil
}

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
