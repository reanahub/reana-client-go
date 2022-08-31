/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package root provides the root command with all the REANA client functionalities as subcommands.
package root

import (
	"os"
	"reanahub/reana-client-go/cmd/close"
	"reanahub/reana-client-go/cmd/delete"
	"reanahub/reana-client-go/cmd/diff"
	"reanahub/reana-client-go/cmd/du"
	"reanahub/reana-client-go/cmd/info"
	"reanahub/reana-client-go/cmd/list"
	"reanahub/reana-client-go/cmd/logs"
	"reanahub/reana-client-go/cmd/ls"
	"reanahub/reana-client-go/cmd/mv"
	"reanahub/reana-client-go/cmd/open"
	"reanahub/reana-client-go/cmd/ping"
	quotaShow "reanahub/reana-client-go/cmd/quota-show"
	"reanahub/reana-client-go/cmd/rm"
	secretsAdd "reanahub/reana-client-go/cmd/secrets-add"
	secretsDelete "reanahub/reana-client-go/cmd/secrets-delete"
	secretsList "reanahub/reana-client-go/cmd/secrets-list"
	"reanahub/reana-client-go/cmd/start"
	"reanahub/reana-client-go/cmd/status"
	"reanahub/reana-client-go/cmd/version"
	"reanahub/reana-client-go/pkg/validator"

	"github.com/spf13/pflag"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type options struct {
	logLevel string
}

// NewCmd creates a new root command, responsible for creating all the other subcommands and
// setting up the logger and persistent flags.
func NewCmd() *cobra.Command {
	o := &options{}
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
	cmd.AddCommand(version.NewCmd())
	cmd.AddCommand(ping.NewCmd())
	cmd.AddCommand(info.NewCmd())
	cmd.AddCommand(list.NewCmd())
	cmd.AddCommand(du.NewCmd())
	cmd.AddCommand(logs.NewCmd())
	cmd.AddCommand(status.NewCmd())
	cmd.AddCommand(ls.NewCmd())
	cmd.AddCommand(diff.NewCmd())
	cmd.AddCommand(quotaShow.NewCmd())
	cmd.AddCommand(open.NewCmd())
	cmd.AddCommand(close.NewCmd())
	cmd.AddCommand(start.NewCmd())
	cmd.AddCommand(delete.NewCmd())
	cmd.AddCommand(rm.NewCmd())
	cmd.AddCommand(mv.NewCmd())
	cmd.AddCommand(secretsAdd.NewCmd())
	cmd.AddCommand(secretsList.NewCmd())
	cmd.AddCommand(secretsDelete.NewCmd())

	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
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
