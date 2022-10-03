/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package fileutils gives extra utils to work with the filesystem.
package fileutils

import (
	"os"
	"path/filepath"
)

// CreateFile provides a way to create a new file ensuring the file path is present.
func CreateFile(name string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(name), os.ModePerm); err != nil {
		return nil, err
	}
	return os.Create(name)
}
