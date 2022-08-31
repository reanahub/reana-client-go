/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package version provides the command to show the version of the client.
package version

import (
	"reanahub/reana-client-go/pkg/config"

	"github.com/spf13/cobra"
)

const description = `
Show version.

The ` + "``version``" + ` command shows REANA client version.
`

// NewCmd creates a command to show the version of the client.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version.",
		Long:  description,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(config.Version)
		},
	}

	return cmd
}
