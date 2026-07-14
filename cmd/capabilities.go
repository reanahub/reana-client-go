/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"

	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
)

// workflowSpecificationBundlesCapability is advertised by servers that load and
// validate an uploaded workflow specification bundle themselves. Released
// servers only accept the retired client-serialized JSON protocol and omit it.
const workflowSpecificationBundlesCapability = "workflow-specification-bundles-v1"

// requireSpecificationBundleSupport refuses a server that cannot accept a
// workflow specification bundle.
//
// The capability is read from the unauthenticated ping operation so the check
// works before authentication, and support is decided by the advertised
// capability rather than by comparing version strings. Callers must invoke this
// before gathering, archiving or uploading anything, so an incompatible pairing
// fails with an actionable message instead of a wire-level error mid-transfer.
func requireSpecificationBundleSupport(api *client.API) error {
	response, err := api.Operations.Ping(operations.NewPingParamsWithTimeout(
		controlOperationTimeout,
	))
	if err != nil {
		return fmt.Errorf("could not check REANA server capabilities: %w", err)
	}

	payload := response.GetPayload()
	if payload != nil {
		for _, capability := range payload.APICapabilities {
			if capability == workflowSpecificationBundlesCapability {
				return nil
			}
		}
	}

	version := ""
	if payload != nil && payload.ReanaServerVersion != "" {
		version = fmt.Sprintf(" (version %s)", payload.ReanaServerVersion)
	}
	return fmt.Errorf(
		"the connected REANA server%s does not support the server-side workflow "+
			"specification validation protocol used by this client (missing %q); "+
			"please upgrade the REANA cluster, or use a REANA client release that "+
			"matches it",
		version,
		workflowSpecificationBundlesCapability,
	)
}

// bundleCapableAPIClient returns a control-plane API client only when the
// connected server implements the specification-bundle protocol.
func bundleCapableAPIClient(token string) (*client.API, error) {
	api, err := client.ControlAPIClient(token)
	if err != nil {
		return nil, err
	}
	if err := requireSpecificationBundleSupport(api.API); err != nil {
		return nil, err
	}
	return api.API, nil
}
