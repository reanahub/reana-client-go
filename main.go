/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package main

import (
	"fmt"
	"os"
	"reanahub/reana-client-go/cmd"
	"reanahub/reana-client-go/utils"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	err := rootCmd.Execute()

	if err != nil {
		err := utils.HandleApiError(err)
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
