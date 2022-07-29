/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"os"
	"reanahub/reana-client-go/validation"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reana-client",
		Short:        "REANA client for interacting with REANA server.",
		Long:         "REANA client for interacting with REANA server.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd)
		},
	}

	cmd.SetOut(os.Stdout)

	cmd.PersistentFlags().StringP("loglevel", "l", "WARNING", "Sets log level [DEBUG|INFO|WARNING]")

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

func runE(cmd *cobra.Command) error {
	logLevelFlag, _ := cmd.Flags().GetString("loglevel")

	if err := addLogger(logLevelFlag); err != nil {
		return err
	}
	return nil
}

func addLogger(logLevelFlag string) error {
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
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.1234",
	})
	return nil
}
