/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type quotaPeriodInfo struct {
	PeriodMonths *int64
	StartDate    string
	EndDate      string
}

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
	if cpuResource.QuotaPeriodMonths != nil && *cpuResource.QuotaPeriodMonths > 0 {
		periodMonths = *cpuResource.QuotaPeriodMonths
	}

	startDate, endDate := getQuotaPeriodDateRange(cpuResource)
	return quotaPeriodInfo{
		PeriodMonths: &periodMonths,
		StartDate:    startDate,
		EndDate:      endDate,
	}
}
