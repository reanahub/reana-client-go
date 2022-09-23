/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/workflows"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

const uploadDesc = `
Upload files and directories to workspace.

The ` + "``upload``" + ` command allows to upload workflow input files and
directories. The SOURCES argument can be repeated and specifies which files
and directories are to be uploaded, see examples below. The default
behaviour is to upload all input files and directories specified in the
reana.yaml file.

Examples:

  $ reana-client upload -w myanalysis.42

  $ reana-client upload -w myanalysis.42 code/mycode.py
`

type uploadOptions struct {
	token    string
	workflow string
}

// newUploadCmd creates a command to upload files and directories to workspace.
func newUploadCmd() *cobra.Command {
	o := &uploadOptions{}

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload files and directories to workspace.",
		Long:  uploadDesc,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd, args)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&o.token, "access-token", "t", "", "Access token of the current user.")
	f.StringVarP(
		&o.workflow,
		"workflow",
		"w", "",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)

	return cmd
}

// TODO: filter files based on .gitignore and .reanaignore

func (o *uploadOptions) run(cmd *cobra.Command, args []string) error {
	var inputPaths []string

	if len(args) > 0 {
		// upload files and directories specified in arguments.
		inputPaths = args
	} else {
		// upload all input files and directories specified in the reana.yaml file.
		spec, err := workflows.GetWorkflowSpecification(o.token, o.workflow)
		if err != nil {
			return err
		}
		inputFiles := spec.Specification.Inputs.Files
		inputDirs := spec.Specification.Inputs.Directories
		if err := o.validateInputs(inputFiles, inputDirs); err != nil {
			return err
		}
		inputPaths = append(inputFiles, inputDirs...)
	}

	files, err := o.collectFiles(cmd, inputPaths)
	if err != nil {
		return err
	}

	for _, file := range files {
		_, err := workflows.UploadFile(o.token, o.workflow, file)
		if err != nil {
			return err
		}
		displayer.DisplayMessage(
			fmt.Sprintf("File %s was successfully uploaded.", file),
			displayer.Success,
			false,
			cmd.OutOrStdout(),
		)
	}
	return nil
}

func (o *uploadOptions) collectFiles(cmd *cobra.Command, inputPaths []string) ([]string, error) {
	log.Debugf("Traverse all the input paths to collect files which needs to be uploaded")
	log.Debugf("paths: %s", strings.Join(inputPaths, ", "))
	var files []string
	for _, dir := range inputPaths {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Filter out directories and symlinks
			if !info.Mode().IsRegular() {
				if !info.IsDir() {
					displayer.DisplayMessage(
						fmt.Sprintf("Ignoring symlink %s", path),
						displayer.Info,
						false,
						cmd.OutOrStdout(),
					)
				}
				return nil
			}
			if !slices.Contains(files, path) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	log.Debugf("Collected files:")
	log.Debugf("files: %s", strings.Join(files, ", "))
	return files, nil
}

func (o *uploadOptions) validateInputs(files, dirs []string) error {
	for _, file := range files {
		pathInfo, err := os.Stat(file)
		if err != nil {
			return err
		}
		if pathInfo.IsDir() {
			return fmt.Errorf("found directory in `inputs.files`: %s", file)
		}
	}
	for _, dir := range dirs {
		pathInfo, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !pathInfo.IsDir() {
			return fmt.Errorf("found file in `inputs.directories`: %s", dir)
		}
	}
	return nil
}
