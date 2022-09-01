/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package cmd provides all the commands for interacting with the REANA server.
package cmd

import (
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
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
func NewRootCmd() (*cobra.Command, error) {
	o := &rootOptions{}
	viper := viper.New()
	if err := setupViper(viper); err != nil {
		return nil, err
	}
	api, err := client.NewApiClient(viper)
	if err != nil {
		return nil, fmt.Errorf("error while creating api client:\n%s", err.Error())
	}

	cmd := &cobra.Command{
		Use:           "reana-client",
		Short:         "REANA client for interacting with REANA server.",
		Long:          "REANA client for interacting with REANA server.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd, viper)
		},
	}

	cmd.SetOut(os.Stdout)

	cmd.PersistentFlags().
		StringVarP(&o.logLevel, "loglevel", "l", "WARNING", "Sets log level [DEBUG|INFO|WARNING]")

	// Add commands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newPingCmd(api, viper))
	cmd.AddCommand(newInfoCmd(api))
	cmd.AddCommand(newListCmd(api, viper))
	cmd.AddCommand(newDuCmd(api))
	cmd.AddCommand(newOpenCmd(api, viper))
	cmd.AddCommand(newCloseCmd(api))
	cmd.AddCommand(newLogsCmd(api))
	cmd.AddCommand(newStatusCmd(api))
	cmd.AddCommand(newLsCmd(api, viper))
	cmd.AddCommand(newDiffCmd(api))
	cmd.AddCommand(newQuotaShowCmd(api))
	cmd.AddCommand(newDeleteCmd(api))
	cmd.AddCommand(newStartCmd(api, viper))
	cmd.AddCommand(newSecretsAddCmd(api))
	cmd.AddCommand(newSecretsListCmd(api))
	cmd.AddCommand(newSecretsDeleteCmd(api))
	cmd.AddCommand(newRmCmd(api))
	cmd.AddCommand(newMvCmd(api))

	return cmd, nil
}

func (o *rootOptions) run(cmd *cobra.Command, viper *viper.Viper) error {
	if err := setupLogger(o.logLevel); err != nil {
		return err
	}

	if err := validateFlags(cmd, viper); err != nil {
		return err
	}

	logCmdFlags(cmd)
	return nil
}

// validateFlags validates access token, server URL and workflow flag values.
func validateFlags(cmd *cobra.Command, viper *viper.Viper) error {
	token := cmd.Flags().Lookup("access-token")
	serverURL := viper.GetString("server-url")
	workflow := cmd.Flags().Lookup("workflow")

	if token != nil {
		if err := bindViperToCmdFlag(token, viper); err != nil {
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

		if err := bindViperToCmdFlag(workflow, viper); err != nil {
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
func setupViper(viper *viper.Viper) error {
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
func bindViperToCmdFlag(f *pflag.Flag, viper *viper.Viper) error {
	if f != nil && !f.Changed && viper.IsSet(f.Name) {
		value := viper.GetString(f.Name)
		if err := f.Value.Set(value); err != nil {
			return err
		}
	}
	return nil
}
