/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package commandgroups allows to override default Cobra usage template with grouped commands.
package commandgroups

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type CommandGroup struct {
	Message  string
	Commands []*cobra.Command
}

type CommandGroups []CommandGroup

// Add adds grouped commands to a specific Cobra command.
func (g CommandGroups) Add(c *cobra.Command) {
	for _, group := range g {
		c.AddCommand(group.Commands...)
	}
}

// SetUsageTemplate sets usage template with command groups for a specific Cobra command.
func (g CommandGroups) SetUsageTemplate(c *cobra.Command) {
	cobra.AddTemplateFunc("commandGroups", func(cmd *cobra.Command) string {
		return cmdGroupsString(g)
	})
	c.SetUsageTemplate(usageTemplate)
}

// cmdGroupsString formats command groups to a string.
func cmdGroupsString(g CommandGroups) string {
	groups := []string{}
	for _, cmdGroup := range g {
		cmds := []string{cmdGroup.Message}
		for _, cmd := range cmdGroup.Commands {
			if cmd.IsAvailableCommand() {
				cmds = append(cmds, "  "+rpad(cmd.Name(), cmd.NamePadding())+"   "+cmd.Short)
			}
		}
		groups = append(groups, strings.Join(cmds, "\n"))
	}
	return strings.Join(groups, "\n\n")
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{commandGroups .}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
