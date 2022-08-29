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
	"os"
	"reanahub/reana-client-go/pkg/config"
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

// ValidateAtLeastOne verifies if the given FlagSet has at least one of the flags given in options changed.
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

// ValidateInputParameters compares input parameters to the given original parameters in reana.yaml.
// Returns the validated parameters and a slice with the errors detected.
func ValidateInputParameters(
	inputParams map[string]string,
	originalParams map[string]any,
) (map[string]string, []error) {
	var errorList []error
	validatedParams := make(map[string]string)
	for param, value := range inputParams {
		_, inOriginalParams := originalParams[param]
		if inOriginalParams {
			validatedParams[param] = value
		} else {
			errorList = append(errorList, fmt.Errorf("given parameter - %s, is not in reana.yaml", param))
		}
	}
	return validatedParams, errorList
}

// ValidateOperationalOptions verifies if options are valid according to the available ones specified in config.
// Returns the validated options, including any necessary translations.
func ValidateOperationalOptions(
	workflowType string,
	options map[string]string,
) (map[string]string, error) {
	validatedOptions := make(map[string]string)
	for option, value := range options {
		translationPerType, validOption := config.AvailableOperationalOptions[option]
		if !validOption {
			return nil, fmt.Errorf("operational option '%s' not supported", option)
		}
		translation, validType := translationPerType[workflowType]
		if !validType {
			return nil, fmt.Errorf(
				"operational option '%s' not supported for %s workflows",
				option,
				workflowType,
			)
		}
		validatedOptions[translation] = value
	}
	return validatedOptions, nil
}

// ValidateFile verifies if the file in the given path exists, is readable and if it isn't a directory.
func ValidateFile(path string) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file '%s' does not exist", path)
	}
	if os.IsPermission(err) {
		return fmt.Errorf("file '%s' is not readable", path)
	}
	if err != nil {
		return err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("file '%s' is a directory", path)
	}

	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}
