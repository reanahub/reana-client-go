/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"github.com/spf13/cobra"
)

const version = "v0.0.0-alpha.1"

const versionDesc = `
Show version.

The ` + "``version``" + ` command shows REANA client version.
`

// newVersionCmd creates a command to show the version of the client.
func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version.",
		Long:  versionDesc,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version)
		},
	}

	return cmd
}
