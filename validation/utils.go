/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package validation

import (
	"fmt"
	"os"
	"strings"
)

func ValidateAccessToken(token string) {
	if strings.TrimSpace(token) == "" {
		fmt.Println("Error: Please provide your access token by using the -t/--access-token flag, or by setting the REANA_ACCESS_TOKEN environment variable.")
		os.Exit(1)
	}
}

func ValidateServerURL(serverURL string) {
	if strings.TrimSpace(serverURL) == "" {
		fmt.Println("Error: Please set REANA_SERVER_URL environment variable.")
		os.Exit(1)
	}
}
