/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"

	"reanahub/reana-client-go/pkg/auth"

	"github.com/spf13/cobra"
)

func newServerAddCmd() *cobra.Command {
	var verify, noVerify bool
	cmd := &cobra.Command{
		Use:   "server-add URL",
		Short: "Save a server without authenticating or selecting it.",
		Long:  "Save a server without authenticating or selecting it. Verification is enabled by default. Existing records are not overwritten; use login --server URL to authenticate and update their TLS settings.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("tls-verify") &&
				cmd.Flags().Changed("no-tls-verify") {
				return fmt.Errorf(
					"--tls-verify and --no-tls-verify cannot be used together",
				)
			}
			var choice *bool
			if cmd.Flags().Changed("tls-verify") {
				choice = &verify
			}
			if cmd.Flags().Changed("no-tls-verify") {
				verify = !noVerify
				choice = &verify
			}
			store, err := auth.NewStore()
			if err != nil {
				return err
			}
			server, err := store.AddServer(args[0], choice)
			if err != nil {
				return err
			}
			cmd.Printf("Saved REANA server: %s (not selected)\n", server)
			return nil
		},
	}
	cmd.Flags().
		BoolVar(&verify, "tls-verify", false, "Save certificate verification as enabled.")
	cmd.Flags().
		BoolVar(&noVerify, "no-tls-verify", false, "Save certificate verification as disabled.")
	return cmd
}

func newServerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server-list",
		Short: "List saved server connections.",
		Long:  "List saved server connections. Show the current selection and effective TLS verification. This command does not contact servers or refresh credentials.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := auth.NewStore()
			if err != nil {
				return err
			}
			servers, err := store.ListServers()
			if err != nil {
				return err
			}
			if len(servers) == 0 {
				cmd.Println("No saved REANA servers.")
			}
			for _, server := range servers {
				marker := " "
				if server.Active {
					marker = "*"
				}
				cmd.Printf(
					"%s %s  TLS verification: %s\n",
					marker,
					server.URL,
					server.TLSVerification,
				)
			}
			return nil
		},
	}
}

func newServerUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server-use URL",
		Short: "Select a saved server without authenticating.",
		Long:  "Select a saved server without authenticating. Credentials and TLS settings are retained. The next command refreshes credentials or asks for login when necessary.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := auth.NewStore()
			if err != nil {
				return err
			}
			server, err := store.UseServer(args[0])
			if err != nil {
				return err
			}
			cmd.Printf("Selected REANA server: %s\n", server)
			return nil
		},
	}
}

func newServerRemoveCmd() *cobra.Command {
	var localOnly bool
	cmd := &cobra.Command{
		Use:   "server-remove URL",
		Short: "Revoke credentials and remove a saved server.",
		Long:  "Revoke credentials and remove a saved server. Revocation failure preserves the record. Use --local-only to forget an unreachable server without revoking its tokens. Removing the selected server leaves no selection; no other server is selected automatically.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := auth.NewManager()
			if err != nil {
				return err
			}
			server, err := manager.RemoveServer(
				cmd.Context(),
				args[0],
				localOnly,
			)
			if err != nil {
				return err
			}
			if server.Unrevoked {
				cmd.PrintErrln("[WARNING] Remote credentials were not revoked.")
			}
			cmd.Printf("Removed saved REANA server: %s\n", server.URL)
			return nil
		},
	}
	cmd.Flags().
		BoolVar(&localOnly, "local-only", false, "Remove locally without revoking credentials at the identity provider.")
	return cmd
}
