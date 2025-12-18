/*
This file is part of REANA.
Copyright (C) 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const completionLongDesc = `Generate shell completion scripts for reana-client-go.

To load completions:

Bash:

  $ source <(reana-client-go completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ reana-client-go completion bash > /etc/bash_completion.d/reana-client-go
  # macOS:
  $ reana-client-go completion bash > $(brew --prefix)/etc/bash_completion.d/reana-client-go

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, add to your .zshrc:
  $ source <(reana-client-go completion zsh)
  $ compdef _reana-client-go reana-client-go

  # Or install to fpath (requires new shell):
  $ reana-client-go completion zsh > "${fpath[1]}/_reana-client-go"

Fish:

  $ reana-client-go completion fish | source

  # To load completions for each session, execute once:
  $ reana-client-go completion fish > ~/.config/fish/completions/reana-client-go.fish

PowerShell:

  PS> reana-client-go completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> reana-client-go completion powershell > reana-client-go.ps1
  # and source this file from your PowerShell profile.
`

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion scripts.",
		Long:                  completionLongDesc,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args: cobra.MatchAll(
			cobra.ExactArgs(1),
			cobra.OnlyValidArgs,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}

	return cmd
}
