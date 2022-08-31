/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package info provides the command to list cluster general information.
package info

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"

	"github.com/spf13/cobra"
)

const description = `
List cluster general information.

The ` + "``info``" + ` command lists general information about the cluster.

Lists all the available workspaces. It also returns the default workspace
defined by the admin.
`

type options struct {
	token      string
	jsonOutput bool
}

// NewCmd creates a command to list cluster general information.
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "info",
		Short: "List cluster general information.",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.BoolVarP(&o.jsonOutput, "json", "", false, "Get output in JSON format.")

	return cmd
}

func (o *options) run(cmd *cobra.Command) error {
	infoParams := operations.NewInfoParams()
	infoParams.SetAccessToken(o.token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	infoResp, err := api.Operations.Info(infoParams)
	if err != nil {
		return err
	}

	p := infoResp.Payload
	if o.jsonOutput {
		err := displayer.DisplayJsonOutput(p, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		if p.ComputeBackends != nil {
			displaySliceItem(cmd, p.ComputeBackends.Title, p.ComputeBackends.Value)
		}
		if p.DefaultKubernetesJobsTimeout != nil {
			displayStringItem(cmd, p.DefaultKubernetesJobsTimeout.Title, &p.DefaultKubernetesJobsTimeout.Value)
		}
		if p.DefaultKubernetesMemoryLimit != nil {
			displayStringItem(cmd, p.DefaultKubernetesMemoryLimit.Title, &p.DefaultKubernetesMemoryLimit.Value)
		}
		if p.DefaultWorkspace != nil {
			displayStringItem(cmd, p.DefaultWorkspace.Title, &p.DefaultWorkspace.Value)
		}
		if p.KubernetesMaxMemoryLimit != nil {
			displayStringItem(cmd, p.KubernetesMaxMemoryLimit.Title, p.KubernetesMaxMemoryLimit.Value)
		}
		if p.MaximumKubernetesJobsTimeout != nil {
			displayStringItem(cmd, p.MaximumKubernetesJobsTimeout.Title, &p.MaximumKubernetesJobsTimeout.Value)
		}
		if p.MaximumWorkspaceRetentionPeriod != nil {
			displayStringItem(cmd, p.MaximumWorkspaceRetentionPeriod.Title, p.MaximumWorkspaceRetentionPeriod.Value)
		}
		if p.WorkspacesAvailable != nil {
			displaySliceItem(cmd, p.WorkspacesAvailable.Title, p.WorkspacesAvailable.Value)
		}
	}
	return nil
}

// displayStringItem displays a nullable string with the given title.
// If valuePtr is nil, prints "None" instead.
func displayStringItem(cmd *cobra.Command, title string, valuePtr *string) {
	value := "None"
	if valuePtr != nil {
		value = *valuePtr
	}
	cmd.Printf("%s: %s\n", title, value)
}

// displaySliceItem displays a slice value with the given title, joined by commas.
func displaySliceItem(cmd *cobra.Command, title string, value []string) {
	cmd.Printf("%s: %s\n", title, strings.Join(value, ", "))
}
