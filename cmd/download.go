/*
This file is part of REANA.
Copyright (C) 2022, 2023 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/displayer"
	"reanahub/reana-client-go/pkg/fileutils"
	"reanahub/reana-client-go/pkg/workflows"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const downloadDesc = `
Download workspace files.

The ` + "``download``" + ` command allows to download workspace files and directories.
By default, the files specified in the workflow specification as outputs are
downloaded. You can also specify the individual files you would like to
download, see examples below.

Examples:

  $ reana-client download # download all output files

  $ reana-client download mydata.tmp outputs/myplot.png

  $ reana-client download -o - data.txt # write data.txt to stdout
`

const outputPathFlagDesc = `Path to the directory where files will be downloaded.
If "-" specified as path, the files will be written to the standard output.`

type downloadOptions struct {
	token      string
	workflow   string
	outputPath string
}

// newDownloadCmd creates a command to download workspace files.
func newDownloadCmd() *cobra.Command {
	o := &downloadOptions{}

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download workspace files.",
		Long:  downloadDesc,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd, args)
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
		&o.workflow,
		"workflow",
		"w",
		"",
		"Name or UUID of the workflow. Overrides value of REANA_WORKON environment variable.",
	)
	f.StringVarP(
		&o.outputPath,
		"output-directory",
		"o",
		"",
		outputPathFlagDesc,
	)

	return cmd
}

func (o *downloadOptions) run(cmd *cobra.Command, args []string) error {
	var downloadPaths []string

	if len(args) > 0 {
		// download files and directories specified in arguments.
		downloadPaths = args
	} else {
		// download all output files and directories specified in the reana.yaml file.
		spec, err := workflows.GetWorkflowSpecification(o.token, o.workflow)
		if err != nil {
			return err
		}
		if outputs := spec.Specification.Outputs; outputs != nil {
			downloadPaths = append(downloadPaths, outputs.Files...)
			downloadPaths = append(downloadPaths, outputs.Directories...)
		}
	}
	log.Debugf("Download paths: %s", strings.Join(downloadPaths, ", "))

	for _, file := range downloadPaths {
		fileName, fileBuf, multipleFilesZipped, err := workflows.DownloadFile(
			o.token,
			o.workflow,
			file,
		)
		if err != nil {
			return err
		}
		if o.outputPath == config.StdoutChar {
			err := o.displayFileContent(
				cmd,
				fileName,
				fileBuf,
				multipleFilesZipped,
			)
			if err != nil {
				return err
			}
		} else {
			err := o.storeFileContent(cmd, fileName, fileBuf)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// displayFileContent writes file(s) content to the standard output.
func (o *downloadOptions) displayFileContent(
	cmd *cobra.Command,
	fileName string,
	fileBuf *bytes.Buffer,
	multipleFilesZipped bool,
) error {
	if multipleFilesZipped {
		// handle zip archive containing multiple files.
		reader := bytes.NewReader(fileBuf.Bytes())
		zipReader, err := zip.NewReader(reader, int64(reader.Len()))
		if err != nil {
			return err
		}
		for _, file := range zipReader.File {
			f, err := file.Open()
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(cmd.OutOrStdout(), f)
			if err != nil {
				return err
			}
		}
	} else {
		// handle single file.
		_, err := io.Copy(cmd.OutOrStdout(), fileBuf)
		if err != nil {
			return err
		}
	}
	return nil
}

// storeFileContent stores file(s) content from a buffer to a file.
func (o *downloadOptions) storeFileContent(
	cmd *cobra.Command,
	fileName string,
	fileBuf *bytes.Buffer,
) error {
	// create a file
	filePath := path.Join(o.outputPath, fileName)
	file, err := fileutils.CreateFile(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// write a buffer to a file
	_, err = io.Copy(file, fileBuf)
	if err != nil {
		return err
	}

	displayer.DisplayMessage(
		fmt.Sprintf("File %s was successfully downloaded.", fileName),
		displayer.Success,
		false,
		cmd.OutOrStdout(),
	)
	return nil
}
