/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package diff provides the command to show diff between two workflows.
package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/datautils"
	"reanahub/reana-client-go/pkg/displayer"

	"github.com/iancoleman/orderedmap"

	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/spf13/cobra"
)

const description = `
Show diff between two workflows.

The ` + "``diff``" + ` command allows to compare two workflows, the workflow_a and
workflow_b, which must be provided as arguments. The output will show the
difference in workflow run parameters, the generated files, the logs, etc.

Examples:

	$ reana-client diff myanalysis.42 myotheranalysis.43

	$ reana-client diff myanalysis.42 myotheranalysis.43 --brief
`

type options struct {
	token     string
	workflowA string
	workflowB string
	brief     bool
	unified   int
}

// NewCmd creates a command to show diff between two workflows.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff between two workflows.",
		Long:  description,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.workflowA = args[0]
			o.workflowB = args[1]
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.BoolVarP(&o.brief, "brief", "q", false, `If not set, differences in the contents of the
files in the two workspaces are shown.`)
	f.IntVarP(
		&o.unified, "unified", "u", 5, "Sets number of context lines for workspace diff output.",
	)

	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
	diffParams := operations.NewGetWorkflowDiffParams()
	diffParams.SetAccessToken(&o.token)
	diffParams.SetWorkflowIDOrNamea(o.workflowA)
	diffParams.SetWorkflowIDOrNameb(o.workflowB)
	diffParams.SetBrief(&o.brief)
	contextLines := fmt.Sprintf("%d", o.unified)
	diffParams.SetContextLines(&contextLines)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	diffResp, err := api.Operations.GetWorkflowDiff(diffParams)
	if err != nil {
		return err
	}

	err = displayPayload(cmd, diffResp.Payload)
	if err != nil {
		return err
	}

	return nil
}

// displayPayload displays the diff payload.
func displayPayload(cmd *cobra.Command, p *operations.GetWorkflowDiffOKBody) error {
	if p.ReanaSpecification != "" {
		specificationDiff := orderedmap.New()
		err := json.Unmarshal([]byte(p.ReanaSpecification), &specificationDiff)
		if err != nil {
			return err
		}

		// Rename section workflow to specification
		val, hasWorkflow := specificationDiff.Get("workflow")
		if hasWorkflow {
			specificationDiff.Set("specification", val)
			specificationDiff.Delete("workflow")
		}
		equalSpecification := true
		for _, section := range specificationDiff.Keys() {
			// Convert diff to a slice of strings
			sectionDiffs, _ := specificationDiff.Get(section)
			linesInterface, ok := sectionDiffs.([]any)
			if !ok {
				return fmt.Errorf("expected diff to be an array, got %v", sectionDiffs)
			}
			lines := make([]string, 0, len(linesInterface))
			for _, line := range linesInterface {
				lineString, ok := line.(string)
				if !ok {
					return fmt.Errorf("expected diff line to be a string, got %v", line)
				}
				lines = append(lines, lineString)
			}

			if len(lines) != 0 {
				equalSpecification = false
				displayer.PrintColorable(
					fmt.Sprintf("%s Differences in workflow %s\n", config.LeadingMark, section),
					cmd.OutOrStdout(),
					text.FgYellow,
					text.Bold,
				)
				printDiff(lines, cmd.OutOrStdout())
			}
		}
		if equalSpecification {
			displayer.PrintColorable(
				fmt.Sprintf("%s No differences in REANA specifications.\n", config.LeadingMark),
				cmd.OutOrStdout(),
				text.FgYellow,
				text.Bold,
			)
		}
		cmd.Println() // Separation line
	}

	var workspaceDiffRaw string
	err := json.Unmarshal([]byte(p.WorkspaceListing), &workspaceDiffRaw)
	if err != nil {
		return err
	}
	if workspaceDiffRaw != "" {
		workspaceDiff := datautils.SplitLinesNoEmpty(workspaceDiffRaw)

		displayer.PrintColorable(
			fmt.Sprintf("%s Differences in workflow workspace\n", config.LeadingMark),
			cmd.OutOrStdout(),
			text.FgYellow,
			text.Bold,
		)
		printDiff(workspaceDiff, cmd.OutOrStdout())
	}

	return nil
}

// printDiff prints the diff contained int the given lines.
func printDiff(lines []string, out io.Writer) {
	for _, line := range lines {
		lineColor := text.Reset
		switch line[0] {
		case '@':
			lineColor = text.FgCyan
		case '-':
			lineColor = text.FgRed
		case '+':
			lineColor = text.FgGreen
		}

		displayer.PrintColorable(line, out, lineColor)
		fmt.Fprintln(out)
	}
}
