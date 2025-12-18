/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

/*
REANA client for interacting with REANA server.

Use --help for more information.
*/
package main

import (
	"os"
	"reanahub/reana-client-go/cmd"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/errorhandler"

	log "github.com/sirupsen/logrus"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	err := rootCmd.Execute()

	if err != nil {
		log.Debug(err)
		err = errorhandler.HandleApiError(err)
		if err != config.ErrEmpty {
			displayer.DisplayMessage(err.Error(), displayer.Error, false, os.Stderr)
		}
		os.Exit(1)
	}
}
