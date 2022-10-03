/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package fileutils

import (
	"path"
	"testing"
)

func TestCreateFile(t *testing.T) {
	tmpdir := t.TempDir()
	filePath := path.Join(tmpdir, "dir1/dir2/file.txt")

	file, err := CreateFile(filePath)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	got := file.Name()
	if got != filePath {
		t.Errorf("Expected %s, got %s", filePath, got)
	}
}
