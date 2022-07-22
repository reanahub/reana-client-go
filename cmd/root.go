/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reana-client",
		Short:        "REANA client for interacting with REANA server.",
		Long:         "REANA client for interacting with REANA server.",
		SilenceUsage: true,
	}

	cmd.SetOut(os.Stdout)

	cmd.PersistentFlags().BoolP("loglevel", "l", false, "Sets log level [DEBUG|INFO|WARNING]")

	// Add commands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newPingCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDuCmd())
	cmd.AddCommand(newOpenCmd())
	cmd.AddCommand(newCloseCmd())

	return cmd
}
