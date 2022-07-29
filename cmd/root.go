/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"os"
	"reanahub/reana-client-go/utils"
	"reanahub/reana-client-go/validation"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	logLevel string
}

func NewRootCmd() *cobra.Command {
	o := &rootOptions{}
	cmd := &cobra.Command{
		Use:          "reana-client",
		Short:        "REANA client for interacting with REANA server.",
		Long:         "REANA client for interacting with REANA server.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	cmd.SetOut(os.Stdout)

	cmd.PersistentFlags().
		StringVarP(&o.logLevel, "loglevel", "l", "WARNING", "Sets log level [DEBUG|INFO|WARNING]")

	// Add commands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newPingCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDuCmd())
	cmd.AddCommand(newOpenCmd())
	cmd.AddCommand(newCloseCmd())
	cmd.AddCommand(newLogsCmd())

	return cmd
}

func (o *rootOptions) run(cmd *cobra.Command) error {
	if err := setupLogger(o.logLevel); err != nil {
		return err
	}

	utils.LogCmdFlags(cmd)
	return nil
}

func setupLogger(logLevelFlag string) error {
	if err := validation.ValidateChoice(
		logLevelFlag,
		[]string{"DEBUG", "INFO", "WARNING"},
		"loglevel",
	); err != nil {
		return err
	}
	level, err := log.ParseLevel(logLevelFlag)
	if err != nil {
		return err
	}
	log.SetLevel(level)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.1234",
	})
	return nil
}
