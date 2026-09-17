/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package cmd provides all the commands for interacting with the REANA server.
package cmd

import (
	"fmt"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/pkg/auth"
	"reanahub/reana-client-go/pkg/commandgroups"
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
		Use:           "reana-client-go",
		Short:         "REANA client for interacting with REANA server.",
		Long:          "REANA client for interacting with REANA server.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
		PersistentPostRunE: func(*cobra.Command, []string) error {
			if err := stopProfiler(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.SetOut(os.Stdout)

	addProfilerFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().
		StringVarP(&o.logLevel, "loglevel", "l", "WARNING", "Sets log level [DEBUG|INFO|WARNING]")

	// Add commands
	commandGroups := commandgroups.CommandGroups{
		{
			Message: "Quota commands:",
			Commands: []*cobra.Command{
				newQuotaShowCmd(),
			},
		},
		{
			Message: "Configuration commands:",
			Commands: []*cobra.Command{
				newCompletionCmd(),
				newInfoCmd(),
				newLoginCmd(),
				newLogoutCmd(),
				newPingCmd(),
				newVersionCmd(),
			},
		},
		{
			Message: "Workflow management commands:",
			Commands: []*cobra.Command{
				newCreateCmd(),
				newDiffCmd(),
				newDeleteCmd(),
				newListCmd(),
			},
		},
		{
			Message: "Workflow execution commands:",
			Commands: []*cobra.Command{
				newRunCmd(),
				newValidateCmd(),
				newStopCmd(),
				newRestartCmd(),
				newLogsCmd(),
				newStartCmd(),
				newStatusCmd(),
			},
		},
		{
			Message: "Workflow run test commands:",
			Commands: []*cobra.Command{
				newTestCmd(),
			},
		},
		{
			Message: "Workflow sharing commands:",
			Commands: []*cobra.Command{
				newShareAddCmd(),
				newShareRemoveCmd(),
				newShareStatusCmd(),
			},
		},
		{
			Message: "Workspace interactive commands:",
			Commands: []*cobra.Command{
				newOpenCmd(),
				newCloseCmd(),
			},
		},
		{
			Message: "Workspace file management commands:",
			Commands: []*cobra.Command{
				newDownloadCmd(),
				newUploadCmd(),
				newDuCmd(),
				newLsCmd(),
				newRmCmd(),
				newPruneCmd(),
				newMvCmd(),
			},
		},
		{
			Message: "Workspace file retention commands:",
			Commands: []*cobra.Command{
				newRetentionRulesListCmd(),
			},
		},
		{
			Message: "Secret management commands:",
			Commands: []*cobra.Command{
				newSecretsAddCmd(),
				newSecretsListCmd(),
				newSecretsDeleteCmd(),
			},
		},
	}
	commandGroups.Add(cmd)
	commandGroups.SetUsageTemplate(cmd)
	return cmd
}

func (o *rootOptions) run(cmd *cobra.Command) error {
	auth.ResetTLSWarning()
	client.SetAuthManager(nil)
	parent := cmd.Parent()
	if cmd.Name() != "help" && cmd.Name() != "version" &&
		cmd.Name() != "completion" &&
		(parent == nil || parent.Name() != "completion") {
		if err := auth.CheckRetiredEnvironment(); err != nil {
			return err
		}
	}
	if err := setupProfiler(); err != nil {
		return err
	}
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
		if tokenValue != "" {
			if !token.Changed && os.Getenv("REANA_ACCESS_TOKEN") != "" &&
				!auth.IsJWT(tokenValue) {
				return &auth.AuthenticationError{
					Message: "REANA_ACCESS_TOKEN must contain a JWT. REANA 0.95 uses OIDC login; unset an old REANA 0.9 token and run `reana-client-go login`, or provide a valid JWT.",
				}
			}
		}
		manager, err := auth.NewManager()
		if err != nil {
			return err
		}
		if serverURL == "" {
			serverURL, err = manager.Store.ActiveServer()
			if err != nil {
				return err
			}
		}
		if serverURL == "" {
			return &auth.AuthenticationError{Message: auth.NoServerMessage}
		}
		serverURL, err = auth.NormalizeServerURL(serverURL)
		if err != nil {
			return err
		}
		viper.Set("server-url", serverURL)
		client.SetAuthManager(manager)
		if _, err := manager.TLSStatus(serverURL); err != nil {
			return err
		}
		if tokenValue == "" {
			if _, err := manager.AccessToken(cmd.Context(), serverURL); err != nil {
				return auth.ForServer(serverURL, "saved login", err)
			}
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
	log.SetFormatter(&cliLogFormatter{TextFormatter: log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.1234",
	}})
	return nil
}

// logCmdFlags logs all the flags set in the given command.
func logCmdFlags(cmd *cobra.Command) {
	log.Debugf("command: %s", cmd.CalledAs())
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name == "access-token" {
			log.Debugf("%s: [REDACTED]", f.Name)
			return
		}
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

// Warnings share the Python client's presentation; debug keeps its detail.
type cliLogFormatter struct{ log.TextFormatter }

func (formatter *cliLogFormatter) Format(entry *log.Entry) ([]byte, error) {
	if entry.Level == log.WarnLevel {
		return []byte(fmt.Sprintf("[WARNING] %s\n", entry.Message)), nil
	}
	return formatter.TextFormatter.Format(entry)
}
