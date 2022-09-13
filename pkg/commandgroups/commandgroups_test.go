/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package commandgroups

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRpad(t *testing.T) {
	tests := map[string]struct {
		str     string
		padding int
		want    string
	}{
		"no padding":       {str: "foo", padding: 0, want: "foo"},
		"padding of one":   {str: "foo", padding: 1, want: "foo"},
		"padding of two":   {str: "foo", padding: 2, want: "foo"},
		"padding of three": {str: "foo", padding: 3, want: "foo"},
		"padding of four":  {str: "foo", padding: 4, want: "foo "},
		"padding of five":  {str: "foo", padding: 5, want: "foo  "},
		"empty string":     {str: "", padding: 1, want: " "},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := rpad(test.str, test.padding)
			if got != test.want {
				t.Errorf("Expected `%s`, got `%s`", test.want, got)
			}
		})
	}
}

func getCobraCmd(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}

var cmdGroupsOutput string = `Quota commands:
  quota-show    Show user quota.

Configuration commands:
  info          List cluster general information.
  ping          Check connection to REANA server.
  version       Show version.`

func TestCommandGroups(t *testing.T) {
	commandGroups := CommandGroups{
		{
			Message: "Quota commands:",
			Commands: []*cobra.Command{
				getCobraCmd("quota-show", "Show user quota."),
			},
		},
		{
			Message: "Configuration commands:",
			Commands: []*cobra.Command{
				getCobraCmd("info", "List cluster general information."),
				getCobraCmd("ping", "Check connection to REANA server."),
				getCobraCmd("version", "Show version."),
			},
		},
	}

	t.Run("command groups", func(t *testing.T) {
		got := cmdGroupsString(commandGroups)
		if got != cmdGroupsOutput {
			t.Errorf("Expected:\n`%s`, got:\n`%s`", cmdGroupsOutput, got)
		}
	})
}
