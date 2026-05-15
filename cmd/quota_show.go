/*
This file is part of REANA.
Copyright (C) 2022, 2025 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/validator"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const quotaShowDesc = `
Show user quota.

The ` + "``quota-show``" + ` command displays quota usage for the user.

Examples:

	$ reana-client quota-show --resource disk --report limit

	$ reana-client quota-show --resource disk --report usage

	$ reana-client quota-show --resource disk

	$ reana-client quota-show --resources
`

type quotaResource struct {
	Health             string             `json:"health"`
	Limit              *quotaResourceStat `json:"limit"`
	QuotaPeriodMonths  *int64             `json:"quota_period_months"`
	QuotaPeriodStartAt *string            `json:"quota_period_start_at"`
	Usage              *quotaResourceStat `json:"usage"`

	Stats map[string]quotaResourceStat `json:"-"`
}

type quotaResourceStat struct {
	HumanReadable string  `json:"human_readable"`
	Raw           float64 `json:"raw"`
}

type quotaShowOptions struct {
	token             string
	report            string
	resource          string
	showResources     bool
	humanReadable     bool
	unspecifiedReport bool
}

type quotaPeriodInfo struct {
	PeriodMonths *int64
	StartDate    string
	EndDate      string
}

// newQuotaShowCmd creates a command to show user quota.
func newQuotaShowCmd() *cobra.Command {
	o := &quotaShowOptions{}

	cmd := &cobra.Command{
		Use:   "quota-show",
		Short: "Show user quota.",
		Long:  quotaShowDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validator.ValidateAtLeastOne(
				cmd.Flags(), []string{"resource", "resources"},
			); err != nil {
				return fmt.Errorf("%s\n%s", err.Error(), cmd.UsageString())
			}
			if cmd.Flags().Changed("report") {
				if err := validator.ValidateChoice(
					o.report, config.QuotaReports, "report",
				); err != nil {
					return err
				}
			} else {
				o.unspecifiedReport = true
			}
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
	f.StringVarP(
		&o.report,
		"report",
		"",
		"",
		"Specify quota report type. e.g. limit, usage.",
	)
	f.StringVarP(
		&o.resource,
		"resource",
		"",
		"",
		"Specify quota resource. e.g. disk, memory.",
	)
	f.BoolVarP(
		&o.showResources,
		"resources",
		"",
		false,
		"Print available resources.",
	)
	f.BoolVarP(
		&o.humanReadable,
		"human-readable",
		"h",
		false,
		"Show disk size in human readable format.",
	)
	// Remove -h shorthand
	cmd.PersistentFlags().BoolP("help", "", false, "Help for quota-show")

	return cmd
}

func (o *quotaShowOptions) run(cmd *cobra.Command) error {
	quotaParams := operations.NewGetYouParams()
	quotaParams.SetAccessToken(&o.token)

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	quotaResp, err := api.Operations.GetYou(quotaParams)
	if err != nil {
		return err
	}
	quotaResources, err := parseQuotaInfo(quotaResp.Payload.Quota)
	if err != nil {
		return err
	}

	var availableResources []string
	for resourceName := range quotaResources {
		availableResources = append(availableResources, resourceName)
	}

	if o.showResources {
		cmd.Println(strings.Join(availableResources, "\n"))
		return nil
	}

	resource, isValidResource := quotaResources[o.resource]
	if !isValidResource {
		return fmt.Errorf(
			"resource '%s' is not valid\nAvailable resources are '%s'",
			o.resource,
			strings.Join(availableResources, "', '"),
		)
	}

	report, reportExists := resource.Stats[o.report]
	if o.unspecifiedReport {
		displayQuotaResourceUsage(
			resource.Health,
			resource.Stats["usage"], resource.Stats["limit"],
			formatQuotaPeriodWindow(resource),
			o.humanReadable, cmd.OutOrStdout(),
		)
	} else if !reportExists || report.Raw <= 0 {
		cmd.Printf("No %s.\n", o.report)
	} else if o.humanReadable {
		msg := report.HumanReadable
		if o.resource == "cpu" {
			if window := formatQuotaPeriodWindow(resource); window != "" {
				msg = fmt.Sprintf("%s in the period from %s", msg, window)
			}
		}
		cmd.Println(msg)
	} else {
		cmd.Printf("%.0f\n", report.Raw)
	}

	return nil
}

// NOTE: Keep this month-boundary arithmetic in sync with
// reana_db/utils.py (_add_months), reana-client's _add_months helper,
// and reana-ui's quota period window calculation.
func addMonths(dt time.Time, months int) time.Time {
	year := dt.Year() + int((int(dt.Month())-1+months)/12)
	month := time.Month((int(dt.Month())-1+months)%12 + 1)
	lastDay := time.Date(
		year,
		month+1,
		0,
		dt.Hour(),
		dt.Minute(),
		dt.Second(),
		dt.Nanosecond(),
		dt.Location(),
	).Day()
	day := dt.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(
		year,
		month,
		day,
		dt.Hour(),
		dt.Minute(),
		dt.Second(),
		dt.Nanosecond(),
		dt.Location(),
	)
}

func formatQuotaPeriodDate(dt time.Time) string {
	return dt.Format("2006-01-02")
}

func getQuotaPeriodDateRange(resource quotaResource) (string, string) {
	if resource.QuotaPeriodMonths == nil || resource.QuotaPeriodStartAt == nil {
		return "", ""
	}

	startAt, err := time.Parse(time.RFC3339Nano, *resource.QuotaPeriodStartAt)
	if err != nil {
		log.Debugf(
			"Could not parse server datetime %q as RFC3339: %v",
			*resource.QuotaPeriodStartAt,
			err,
		)
		return "", ""
	}

	endAt := addMonths(startAt, int(*resource.QuotaPeriodMonths))
	return formatQuotaPeriodDate(startAt), formatQuotaPeriodDate(endAt)
}

func formatQuotaPeriodWindow(resource quotaResource) string {
	startDate, endDate := getQuotaPeriodDateRange(resource)
	if startDate == "" || endDate == "" {
		return ""
	}

	return fmt.Sprintf("%s to %s", startDate, endDate)
}

func buildCPUQuotaPeriodInfo(
	quotaResources map[string]quotaResource,
) quotaPeriodInfo {
	cpuResource, ok := quotaResources["cpu"]
	if !ok {
		return quotaPeriodInfo{}
	}

	periodMonths := int64(0)
	if cpuResource.QuotaPeriodMonths != nil &&
		*cpuResource.QuotaPeriodMonths > 0 {
		periodMonths = *cpuResource.QuotaPeriodMonths
	}

	startDate, endDate := getQuotaPeriodDateRange(cpuResource)
	return quotaPeriodInfo{
		PeriodMonths: &periodMonths,
		StartDate:    startDate,
		EndDate:      endDate,
	}
}

// displayQuotaResourceUsage displays the resource usage of the quotas, using its usage and limit.
func displayQuotaResourceUsage(
	health string,
	usage, limit quotaResourceStat,
	periodWindow string,
	humanReadable bool,
	out io.Writer,
) {
	var limitInfo string
	var includeHealthColor bool
	if limit.Raw > 0 {
		percentage := fmt.Sprintf("%.0f%%", (usage.Raw/limit.Raw)*100)
		if humanReadable {
			limitInfo = fmt.Sprintf(
				"out of %s used (%s)",
				limit.HumanReadable,
				percentage,
			)
		} else {
			limitInfo = fmt.Sprintf("out of %.0f used (%s)", limit.Raw, percentage)
		}
		includeHealthColor = health != ""
	} else {
		limitInfo = "used"
		includeHealthColor = false
	}

	color := text.Reset
	if includeHealthColor {
		color = displayer.ResourceHealthToColor[health]
	}

	var usageMsg string
	if humanReadable {
		usageMsg = usage.HumanReadable
	} else {
		usageMsg = fmt.Sprintf("%.0f", usage.Raw)
	}
	usageMsg = fmt.Sprintf("%s %s", usageMsg, limitInfo)
	if periodWindow != "" {
		// NOTE: This splice relies on the formatted usage message ending with
		// a trailing parenthetical when percentage output is present.
		if strings.HasSuffix(usageMsg, ")") {
			usageMsg = strings.TrimSuffix(usageMsg, ")") +
				fmt.Sprintf(" in the period from %s)", periodWindow)
		} else {
			usageMsg = fmt.Sprintf("%s in the period from %s", usageMsg, periodWindow)
		}
	}
	usageMsg += "\n"
	displayer.PrintColorable(usageMsg, out, color)
}

// parseQuotaInfo parses the quota payload to a map of quotaResource values, with the resource names as keys.
// Necessary because all the resources implement different structs in the swagger API.
func parseQuotaInfo(
	quotaBody *operations.GetYouOKBodyQuota,
) (map[string]quotaResource, error) {
	var rawResourcesInfo map[string]json.RawMessage
	quotaBinary, err := quotaBody.MarshalBinary()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(quotaBinary, &rawResourcesInfo)
	if err != nil {
		return nil, err
	}

	quotaResources := make(map[string]quotaResource)
	for resourceName, rawInfo := range rawResourcesInfo {
		var resourceInfo quotaResource
		err = json.Unmarshal(rawInfo, &resourceInfo)
		if err != nil {
			return nil, err
		}
		resourceInfo.Stats = map[string]quotaResourceStat{}
		if resourceInfo.Limit != nil {
			resourceInfo.Stats["limit"] = *resourceInfo.Limit
		}
		if resourceInfo.Usage != nil {
			resourceInfo.Stats["usage"] = *resourceInfo.Usage
		}

		quotaResources[resourceName] = resourceInfo
	}

	return quotaResources, nil
}
