/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/specbundle"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// localImageCheckTimeout bounds a single local docker/podman pull+inspect.
const localImageCleanupTimeout = 30 * time.Second

var localImageCheckTimeout = 10 * time.Minute

const validateDesc = `
Validate workflow specification file.

The ` + "``validate``" + ` command validates the reana.yaml specification file.
Loading and validation are performed on the REANA server (sandboxed for
Snakemake/CWL/Yadage workflows), so no workflow engines are needed locally.

Examples:

  $ reana-client validate -f reana.yaml
`

// validationEntry is a single error or warning returned by the server.
type validationEntry struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// imageEnvironment binds an image to the effective identity of the step using
// it.
type imageEnvironment struct {
	Image      string `json:"image"`
	RuntimeUID int    `json:"runtime_uid"`
	RuntimeGID int    `json:"runtime_gid"`
}

type validationReport struct {
	Valid    bool              `json:"valid"`
	Errors   []validationEntry `json:"errors"`
	Warnings []validationEntry `json:"warnings"`
	Message  string            `json:"message"`
	// Populated by the server when --environments is requested, so the client
	// can run the deep --pull checks locally without parsing the spec itself.
	Environments          []imageEnvironment `json:"environments"`
	EnvironmentsTruncated bool               `json:"environments_truncated"`
}

type validateOptions struct {
	token              string
	serverURL          string
	file               string
	environments       bool
	pull               bool
	serverCapabilities bool
}

// newValidateCmd creates a command to validate a workflow specification file.
func newValidateCmd() *cobra.Command {
	o := &validateOptions{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate workflow specification file.",
		Long:  validateDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.serverURL = viper.GetString("server-url")
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
		&o.file,
		"file",
		"f",
		"reana.yaml",
		"REANA specification file describing the workflow to validate. [default=reana.yaml]",
	)
	f.BoolVar(
		&o.environments,
		"environments",
		false,
		"If set, check all runtime environments specified in REANA "+
			"specification file. [default=False]",
	)
	f.BoolVar(
		&o.pull,
		"pull",
		false,
		"If set, try to pull remote environment image from registry to perform "+
			"validation locally. Requires --environments flag. [default=False]",
	)
	f.BoolVar(
		&o.serverCapabilities,
		"server-capabilities",
		false,
		"Deprecated and no longer needed: server capabilities (compute "+
			"backends, workspace, vetted images) are now always validated "+
			"server-side. [default=False]",
	)

	return cmd
}

func (o *validateOptions) run(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	if o.pull && !o.environments {
		return fmt.Errorf("`--pull` flag requires `--environments` flag")
	}

	if o.serverCapabilities {
		displayer.DisplayMessage(
			"Server capabilities (compute backends, workspace, vetted images) "+
				"are now always validated server-side; the "+
				"`--server-capabilities` flag is no longer needed and has no effect.",
			displayer.Warning,
			false,
			out,
		)
	}

	displayer.DisplayMessage(
		fmt.Sprintf("Verifying REANA specification file... %s", o.file),
		displayer.Info,
		false,
		out,
	)

	// Refuse an incompatible server before any bundle is built or uploaded.
	api, err := bundleCapableAPIClient(o.token)
	if err != nil {
		return err
	}

	members, err := specbundle.Gather(o.file)
	if err != nil {
		return err
	}
	bundle, err := specbundle.Archive(members)
	if err != nil {
		return err
	}
	defer func() { _ = bundle.Close() }()

	// --environments requests offline image-tag checks and runtime identities;
	// --pull's deep checks run locally below using those identities.
	params := operations.NewValidateWorkflowSpecificationParamsWithTimeout(
		controlOperationTimeout,
	)
	params.SetBundle(bundle)
	if o.environments {
		params.SetEnvironments(&o.environments)
	}
	response, err := api.Operations.ValidateWorkflowSpecification(params, nil)
	if err != nil {
		return err
	}

	body, err := json.Marshal(response.GetPayload())
	if err != nil {
		return fmt.Errorf("could not read server validation response: %w", err)
	}
	var report validationReport
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("could not read server validation response: %w", err)
	}

	for _, warning := range report.Warnings {
		displayer.DisplayMessage(warning.Message, displayer.Warning, true, out)
	}
	if o.pull {
		for _, message := range checkImagesLocally(report.Environments) {
			displayer.DisplayMessage(message, displayer.Warning, true, out)
		}
	}
	if report.Valid {
		message := "Valid REANA specification file."
		if o.pull && report.EnvironmentsTruncated {
			message = "Valid REANA specification file; local environment checks were incomplete."
		}
		displayer.DisplayMessage(message, displayer.Success, false, out)
		return nil
	}
	for _, validationError := range report.Errors {
		displayer.DisplayMessage(
			validationError.Message,
			displayer.Error,
			true,
			out,
		)
	}
	return fmt.Errorf("%s is not a valid REANA specification", o.file)
}

// localContainerCLI returns an available container CLI (docker/podman) or "".
func localContainerCLI() string {
	for _, cli := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(cli); err == nil {
			return cli
		}
	}
	return ""
}

// imageUIDGIDs pulls an image, ignores its configured entrypoint, and
// deliberately runs the image-provided shell and id binaries to read its
// default UID and GIDs.
func imageUIDGIDs(cli, image string) (int, []int, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		localImageCheckTimeout,
	)
	defer cancel()
	// Best-effort refresh; failures are ignored (a locally-built image may not
	// exist in any registry). The run below is the authority -- it uses the local
	// copy, auto-pulls a missing remote image, and fails clearly if absent.
	_ = exec.CommandContext(ctx, cli, "pull", image).Run()
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err != nil {
		return 0, nil, fmt.Errorf("could not allocate container name: %w", err)
	}
	containerName := "reana-validation-" + hex.EncodeToString(identifier)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), localImageCleanupTimeout,
		)
		defer cleanupCancel()
		_ = exec.CommandContext(
			cleanupCtx, cli, "rm", "-f", containerName,
		).Run()
	}()
	out, err := exec.CommandContext(
		ctx,
		cli,
		"run",
		"--name",
		containerName,
		"--rm",
		"--entrypoint",
		"/bin/sh",
		image,
		"-c",
		"id -u && id -G",
	).Output()
	if err != nil {
		detail := err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			detail = strings.TrimSpace(string(exitErr.Stderr))
		}
		return 0, nil, fmt.Errorf("%s", detail)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		return 0, nil, fmt.Errorf("unexpected id output: %q", string(out))
	}
	uid, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, nil, fmt.Errorf("could not parse uid %q", fields[0])
	}
	var gids []int
	for _, field := range fields[1:] {
		if gid, err := strconv.Atoi(field); err == nil {
			gids = append(gids, gid)
		}
	}
	return uid, gids, nil
}

var (
	findLocalContainerCLI = localContainerCLI
	inspectImageUIDGIDs   = imageUIDGIDs
)

// checkImagesLocally pulls each image and warns when its UID/GIDs are
// incompatible with any effective step runtime identity.
func checkImagesLocally(environments []imageEnvironment) []string {
	seen := map[imageEnvironment]bool{}
	var unique []imageEnvironment
	for _, environment := range environments {
		if environment.Image == "" || seen[environment] {
			continue
		}
		seen[environment] = true
		unique = append(unique, environment)
	}
	if len(unique) == 0 {
		return nil
	}

	cli := findLocalContainerCLI()
	if cli == "" {
		return []string{
			"No local container engine (docker/podman) was found, so the " +
				"--pull image checks were skipped.",
		}
	}

	type inspectionResult struct {
		uid  int
		gids []int
		err  error
	}
	inspectedImages := map[string]inspectionResult{}
	var messages []string
	for _, environment := range unique {
		result, inspected := inspectedImages[environment.Image]
		if !inspected {
			result.uid, result.gids, result.err = inspectImageUIDGIDs(
				cli, environment.Image,
			)
			inspectedImages[environment.Image] = result
		}
		if result.err != nil {
			if !inspected {
				messages = append(messages, fmt.Sprintf(
					"Could not pull/inspect image '%s': %s",
					environment.Image,
					result.err,
				))
			}
			continue
		}
		if !containsInt(result.gids, environment.RuntimeGID) {
			messages = append(messages, fmt.Sprintf(
				"Image '%s' user is not a member of GID %d (found %v); files may "+
					"be inaccessible when REANA runs the step as UID %d/GID %d.",
				environment.Image,
				environment.RuntimeGID,
				result.gids,
				environment.RuntimeUID,
				environment.RuntimeGID,
			))
		}
		if result.uid != environment.RuntimeUID {
			messages = append(messages, fmt.Sprintf(
				"Image '%s' default UID is %d but REANA runs steps as UID %d.",
				environment.Image, result.uid, environment.RuntimeUID,
			))
		}
	}
	return messages
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
