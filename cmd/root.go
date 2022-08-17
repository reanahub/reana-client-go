/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package cmd provides all the commands for interacting with the REANA server.
package cmd

import (
	"os"
	"reanahub/reana-client-go/pkg/validator"

	"github.com/spf13/pflag"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type rootOptions struct {
	logLevel string
}

// NewRootCmd creates a new root command, responsible for creating all the other subcommands and
// setting up the logger and persistent flags.
func NewRootCmd() *cobra.Command {
	o := &rootOptions{}
	cmd := &cobra.Command{
		Use:           "reana-client",
		Short:         "REANA client for interacting with REANA server.",
		Long:          "REANA client for interacting with REANA server.",
		SilenceUsage:  true,
		SilenceErrors: true,
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
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newLsCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newQuotaShowCmd())

	return cmd
}

func (o *rootOptions) run(cmd *cobra.Command) error {
	if err := setupLogger(o.logLevel); err != nil {
		return err
	}

	if err := setupViper(); err != nil {
		return err
	}

	if err := validateFlags(cmd); err != nil {
		return err
	}

	logCmdFlags(cmd)
	return nil
}

// validateFlags validates access token, server URL and workflow flag values.
func validateFlags(cmd *cobra.Command) error {
	token := cmd.Flags().Lookup("access-token")
	serverURL := viper.GetString("server-url")
	workflow := cmd.Flags().Lookup("workflow")

	if token != nil {
		if err := bindViperToCmdFlag(token); err != nil {
			return err
		}
		tokenValue := token.Value.String()
		if err := validator.ValidateAccessToken(tokenValue); err != nil {
			return err
		}
		if err := validator.ValidateServerURL(serverURL); err != nil {
			return err
		}
	}
	if workflow != nil {
		properties, ok := workflow.Annotations["properties"]
		optional := ok && slices.Contains(properties, "optional")
		if optional {
			return nil
		}

		if err := bindViperToCmdFlag(workflow); err != nil {
			return err
		}
		workflowValue := workflow.Value.String()
		if err := validator.ValidateWorkflow(workflowValue); err != nil {
			return err
		}
	}
	return nil
}

// setupViper binds environment variable values to the viper keys.
func setupViper() error {
	if err := viper.BindEnv("server-url", "REANA_SERVER_URL"); err != nil {
		return err
	}
	if err := viper.BindEnv("access-token", "REANA_ACCESS_TOKEN"); err != nil {
		return err
	}
	if err := viper.BindEnv("workflow", "REANA_WORKON"); err != nil {
		return err
	}
	return nil
}

// setupLogger validates the logging level flag and configures the logger.
func setupLogger(logLevelFlag string) error {
	if err := validator.ValidateChoice(
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

// logCmdFlags logs all the flags set in the given command.
func logCmdFlags(cmd *cobra.Command) {
	log.Debugf("command: %s", cmd.CalledAs())
	cmd.Flags().Visit(func(f *pflag.Flag) {
		log.Debugf("%s: %s", f.Name, f.Value)
	})
}

// bindViperToCmdFlag applies viper config value to the flag when the flag is not set and viper has a value.
func bindViperToCmdFlag(f *pflag.Flag) error {
	if f != nil && !f.Changed && viper.IsSet(f.Name) {
		value := viper.GetString(f.Name)
		if err := f.Value.Set(value); err != nil {
			return err
		}
	}
	return nil
}
