/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLoginCmd() *cobra.Command {
	var serverURL string
	var headless bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate against REANA Server using OIDC.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if serverURL == "" {
				serverURL = viper.GetString("server-url")
			}
			normalized, err := auth.NormalizeServerURL(serverURL)
			if err != nil {
				return err
			}
			manager, err := auth.NewManager()
			if err != nil {
				return err
			}
			loginContext, cancel := context.WithTimeout(
				cmd.Context(),
				5*time.Minute,
			)
			defer cancel()
			if headless {
				_, err = manager.LoginDevice(
					loginContext,
					normalized,
					func(prompt auth.DevicePrompt) {
						if prompt.VerificationURIComplete != "" {
							cmd.Printf(
								"Open the following URL to authenticate:\n%s\n",
								prompt.VerificationURIComplete,
							)
							return
						}
						cmd.Printf(
							"Open the following URL to authenticate:\n%s\nCode: %s\n",
							prompt.VerificationURI,
							prompt.UserCode,
						)
					},
				)
			} else {
				_, err = manager.LoginBrowser(loginContext, normalized, func(authorizationURL string) {
					cmd.Printf("Opening your browser to authenticate. If it does not open automatically, visit:\n%s\n", authorizationURL)
				}, openBrowser)
			}
			if err != nil {
				return err
			}
			cmd.Printf("Logged in to %s\n", normalized)
			return nil
		},
	}
	cmd.Flags().
		StringVar(&serverURL, "server-url", "", "REANA server URL to authenticate against.")
	cmd.Flags().
		BoolVar(&headless, "headless", false, "Use device login instead of opening a local browser.")
	return cmd
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out from the active REANA Server.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager, err := auth.NewManager()
			if err != nil {
				return err
			}
			serverURL, err := manager.Store.ActiveServer()
			if err != nil {
				return err
			}
			warning, err := manager.Logout(cmd.Context(), serverURL)
			if err != nil {
				return err
			}
			if warning != "" {
				cmd.Printf("Warning: %s\n", warning)
			}
			cmd.Printf("Logged out from %s\n", serverURL)
			return nil
		},
	}
}

func openBrowser(target string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command, args = "open", []string{target}
	case "windows":
		command, args = "rundll32", []string{
			"url.dll,FileProtocolHandler",
			target,
		}
	default:
		command, args = "xdg-open", []string{target}
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("could not find browser opener %q: %w", command, err)
	}
	return exec.Command(path, args...).Start()
}
