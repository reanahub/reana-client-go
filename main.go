/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package main

import (
	"os"
	"reanahub/reana-client-go/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	err := rootCmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
