/*
This file is part of REANA.
Copyright (C) 2022, 2023, 2024, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"strings"

	log "github.com/sirupsen/logrus"

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
func newInfoCmd() *cobra.Command {
	o := &infoOptions{}

	cmd := &cobra.Command{
		Use:   "info",
		Short: "List cluster general information.",
		Long:  infoDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVarP(
		&o.token,
		"access-token",
		"t",
		"",
		"Access token of the current user.",
	)
	f.BoolVarP(&o.jsonOutput, "json", "", false, "Get output in JSON format.")

	return cmd
}

func (o *infoOptions) run(cmd *cobra.Command) error {
	infoParams := operations.NewInfoParams()
	infoParams.SetAccessToken(o.token)
	quotaParams := operations.NewGetYouParams()
	quotaParams.SetAccessToken(&o.token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	infoResp, err := api.Operations.Info(infoParams)
	if err != nil {
		return err
	}
	quotaPeriodInfo := quotaPeriodInfo{}
	quotaResp, err := api.Operations.GetYou(quotaParams)
	if err != nil {
		log.Debugf(
			"Could not enrich cluster info with quota period details: %v",
			err,
		)
	} else {
		quotaResources, err := parseQuotaInfo(quotaResp.Payload.Quota)
		if err != nil {
			log.Debugf("Could not parse quota period details: %v", err)
		} else {
			quotaPeriodInfo = buildCPUQuotaPeriodInfo(quotaResources)
		}
	}

	p := infoResp.Payload
	if o.jsonOutput {
		infoMap, err := buildInfoOutputMap(p, quotaPeriodInfo)
		if err != nil {
			return err
		}
		err = displayer.DisplayJsonOutput(infoMap, cmd.OutOrStdout())
		if err != nil {
			return err
		}
	} else {
		// Fields are ordered alphabetically by JSON key name to match Python client
		if p.ComputeBackends != nil {
			displayInfoSliceItem(cmd, p.ComputeBackends.Title, p.ComputeBackends.Value)
		}
		if p.CwlEngineTool != nil {
			displayInfoStringItem(cmd, p.CwlEngineTool.Title, &p.CwlEngineTool.Value)
		}
		if p.CwlEngineVersion != nil {
			displayInfoStringItem(cmd, p.CwlEngineVersion.Title, &p.CwlEngineVersion.Value)
		}
		if p.DaskEnabled != nil {
			displayInfoStringItem(cmd, p.DaskEnabled.Title, &p.DaskEnabled.Value)
		}
		if p.DaskEnabled != nil && strings.ToLower(p.DaskEnabled.Value) == "true" {
			if p.DaskAutoscalerEnabled != nil {
				displayInfoStringItem(cmd, p.DaskAutoscalerEnabled.Title, &p.DaskAutoscalerEnabled.Value)
			}
			if p.DaskClusterDefaultNumberOfWorkers != nil {
				displayInfoStringItem(cmd, p.DaskClusterDefaultNumberOfWorkers.Title, &p.DaskClusterDefaultNumberOfWorkers.Value)
			}
			if p.DaskClusterDefaultSingleWorkerMemory != nil {
				displayInfoStringItem(cmd, p.DaskClusterDefaultSingleWorkerMemory.Title, &p.DaskClusterDefaultSingleWorkerMemory.Value)
			}
			if p.DaskClusterDefaultSingleWorkerThreads != nil {
				displayInfoStringItem(cmd, p.DaskClusterDefaultSingleWorkerThreads.Title, &p.DaskClusterDefaultSingleWorkerThreads.Value)
			}
			if p.DaskClusterMaxMemoryLimit != nil {
				displayInfoStringItem(cmd, p.DaskClusterMaxMemoryLimit.Title, &p.DaskClusterMaxMemoryLimit.Value)
			}
			if p.DaskClusterMaxNumberOfWorkers != nil {
				displayInfoStringItem(cmd, p.DaskClusterMaxNumberOfWorkers.Title, &p.DaskClusterMaxNumberOfWorkers.Value)
			}
			if p.DaskClusterMaxSingleWorkerMemory != nil {
				displayInfoStringItem(cmd, p.DaskClusterMaxSingleWorkerMemory.Title, &p.DaskClusterMaxSingleWorkerMemory.Value)
			}
			if p.DaskClusterMaxSingleWorkerThreads != nil {
				displayInfoStringItem(cmd, p.DaskClusterMaxSingleWorkerThreads.Title, &p.DaskClusterMaxSingleWorkerThreads.Value)
			}
		}
		if p.DefaultKubernetesCPULimit != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesCPULimit.Title, p.DefaultKubernetesCPULimit.Value)
		}
		if p.DefaultKubernetesCPURequest != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesCPURequest.Title, p.DefaultKubernetesCPURequest.Value)
		}
		if p.DefaultKubernetesJobsTimeout != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesJobsTimeout.Title, &p.DefaultKubernetesJobsTimeout.Value)
		}
		if p.DefaultKubernetesMemoryLimit != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesMemoryLimit.Title, p.DefaultKubernetesMemoryLimit.Value)
		}
		if p.DefaultKubernetesMemoryRequest != nil {
			displayInfoStringItem(cmd, p.DefaultKubernetesMemoryRequest.Title, p.DefaultKubernetesMemoryRequest.Value)
		}
		if p.DefaultWorkspace != nil {
			displayInfoStringItem(cmd, p.DefaultWorkspace.Title, &p.DefaultWorkspace.Value)
		}
		if p.GitlabHost != nil {
			displayInfoStringItem(cmd, p.GitlabHost.Title, &p.GitlabHost.Value)
		}
		if p.InteractiveSessionRecommendedJupyterImages != nil {
			displayInfoSliceItem(cmd, p.InteractiveSessionRecommendedJupyterImages.Title, p.InteractiveSessionRecommendedJupyterImages.Value)
		}
		if p.InteractiveSessionsCustomImageAllowed != nil {
			displayInfoStringItem(cmd, p.InteractiveSessionsCustomImageAllowed.Title, &p.InteractiveSessionsCustomImageAllowed.Value)
		}
		if p.KubernetesMaxCPULimit != nil {
			displayInfoStringItem(cmd, p.KubernetesMaxCPULimit.Title, p.KubernetesMaxCPULimit.Value)
		}
		if p.KubernetesMaxCPURequest != nil {
			displayInfoStringItem(cmd, p.KubernetesMaxCPURequest.Title, p.KubernetesMaxCPURequest.Value)
		}
		if p.KubernetesMaxMemoryLimit != nil {
			displayInfoStringItem(cmd, p.KubernetesMaxMemoryLimit.Title, p.KubernetesMaxMemoryLimit.Value)
		}
		if p.KubernetesMaxMemoryRequest != nil {
			displayInfoStringItem(cmd, p.KubernetesMaxMemoryRequest.Title, p.KubernetesMaxMemoryRequest.Value)
		}
		if p.MaximumInteractiveSessionInactivityPeriod != nil {
			displayInfoStringItem(cmd, p.MaximumInteractiveSessionInactivityPeriod.Title, p.MaximumInteractiveSessionInactivityPeriod.Value)
		}
		if p.MaximumKubernetesJobsTimeout != nil {
			displayInfoStringItem(cmd, p.MaximumKubernetesJobsTimeout.Title, &p.MaximumKubernetesJobsTimeout.Value)
		}
		if p.MaximumWorkspaceRetentionPeriod != nil {
			displayInfoStringItem(cmd, p.MaximumWorkspaceRetentionPeriod.Title, p.MaximumWorkspaceRetentionPeriod.Value)
		}
		if p.SnakemakeEngineVersion != nil {
			displayInfoStringItem(cmd, p.SnakemakeEngineVersion.Title, &p.SnakemakeEngineVersion.Value)
		}
		if p.SupportedWorkflowEngines != nil {
			displayInfoSliceItem(cmd, p.SupportedWorkflowEngines.Title, p.SupportedWorkflowEngines.Value)
		}
		if p.WorkspacesAvailable != nil {
			displayInfoSliceItem(cmd, p.WorkspacesAvailable.Title, p.WorkspacesAvailable.Value)
		}
		if p.YadageEngineAdageVersion != nil {
			displayInfoStringItem(cmd, p.YadageEngineAdageVersion.Title, &p.YadageEngineAdageVersion.Value)
		}
		if p.YadageEnginePacktivityVersion != nil {
			displayInfoStringItem(cmd, p.YadageEnginePacktivityVersion.Title, &p.YadageEnginePacktivityVersion.Value)
		}
		if p.YadageEngineVersion != nil {
			displayInfoStringItem(cmd, p.YadageEngineVersion.Title, &p.YadageEngineVersion.Value)
		}
		displayQuotaPeriodInfo(cmd, quotaPeriodInfo)
	}
	return nil
}

func buildInfoOutputMap(
	payload *operations.InfoOKBody,
	quotaInfo quotaPeriodInfo,
) (map[string]any, error) {
	var infoMap map[string]any
	infoBinary, err := payload.MarshalBinary()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(infoBinary, &infoMap)
	if err != nil {
		return nil, err
	}
	if quotaInfo.PeriodMonths != nil {
		infoMap["cpu_quota_period_months"] = map[string]any{
			"title": "CPU quota period in months",
			"value": *quotaInfo.PeriodMonths,
		}
	}
	if quotaInfo.StartDate != "" {
		infoMap["current_cpu_quota_period_start"] = map[string]string{
			"title": "Current CPU quota period start",
			"value": quotaInfo.StartDate,
		}
	}
	if quotaInfo.EndDate != "" {
		infoMap["current_cpu_quota_period_end"] = map[string]string{
			"title": "Current CPU quota period end",
			"value": quotaInfo.EndDate,
		}
	}
	return infoMap, nil
}

func displayQuotaPeriodInfo(cmd *cobra.Command, quotaInfo quotaPeriodInfo) {
	if quotaInfo.PeriodMonths != nil {
		cmd.Printf("CPU quota period in months: %d\n", *quotaInfo.PeriodMonths)
	}
	if quotaInfo.StartDate != "" {
		cmd.Printf("Current CPU quota period start: %s\n", quotaInfo.StartDate)
	}
	if quotaInfo.EndDate != "" {
		cmd.Printf("Current CPU quota period end: %s\n", quotaInfo.EndDate)
	}
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
