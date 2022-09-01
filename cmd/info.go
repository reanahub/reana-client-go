/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"

	"github.com/spf13/cobra"
)

const infoDesc = `
List cluster general information.

The ` + "``info``" + ` command lists general information about the cluster.

Lists all the available workspaces. It also returns the default workspace
defined by the admin.
`

type infoOptions struct {
	token      string
	jsonOutput bool
}

// newInfoCmd creates a command to list cluster general information.
func newInfoCmd(api *client.API) *cobra.Command {
	o := &infoOptions{}

	cmd := &cobra.Command{
		Use:   "info",
		Short: "List cluster general information.",
		Long:  infoDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd, api)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.BoolVarP(&o.jsonOutput, "json", "", false, "Get output in JSON format.")

	return cmd
}

func (o *infoOptions) run(cmd *cobra.Command, api *client.API) error {
	infoParams := operations.NewInfoParams()
	infoParams.SetAccessToken(o.token)

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
			displayInfoSliceItem(cmd, p.ComputeBackends.Title, p.ComputeBackends.Value)
		}
		if p.DefaultKubernetesJobsTimeout != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesJobsTimeout.Title, &p.DefaultKubernetesJobsTimeout.Value)
		}
		if p.DefaultKubernetesMemoryLimit != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesMemoryLimit.Title, &p.DefaultKubernetesMemoryLimit.Value)
		}
		if p.DefaultWorkspace != nil {
			displayInfoStringItem(cmd, p.DefaultWorkspace.Title, &p.DefaultWorkspace.Value)
		}
		if p.KubernetesMaxMemoryLimit != nil {
			displayInfoStringItem(cmd, p.KubernetesMaxMemoryLimit.Title, p.KubernetesMaxMemoryLimit.Value)
		}
		if p.MaximumKubernetesJobsTimeout != nil {
			displayInfoStringItem(cmd, p.MaximumKubernetesJobsTimeout.Title, &p.MaximumKubernetesJobsTimeout.Value)
		}
		if p.MaximumWorkspaceRetentionPeriod != nil {
			displayInfoStringItem(cmd, p.MaximumWorkspaceRetentionPeriod.Title, p.MaximumWorkspaceRetentionPeriod.Value)
		}
		if p.WorkspacesAvailable != nil {
			displayInfoSliceItem(cmd, p.WorkspacesAvailable.Title, p.WorkspacesAvailable.Value)
		}
	}
	return nil
}

func displayInfoStringItem(cmd *cobra.Command, title string, valuePtr *string) {
	value := "None"
	if valuePtr != nil {
		value = *valuePtr
	}
	cmd.Printf("%s: %s\n", title, value)
}

func displayInfoSliceItem(cmd *cobra.Command, title string, value []string) {
	cmd.Printf("%s: %s\n", title, strings.Join(value, ", "))
}
