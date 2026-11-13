/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

// Package specbundle builds and streams the explicitly declared workflow
// definition snapshot used by server-side validation.
package specbundle

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	canonicalSpecification = "reana.yaml"
	defaultBundleMaxFiles  = 1000
	defaultBundleMaxBytes  = 100 * 1024 * 1024
	defaultBundleMaxPath   = 4096
	bundleMaxDirectories   = 2000
	bundleMaxDepth         = 64
)

var (
	bundleMaxFiles = configuredPositiveInt(
		"REANA_SPEC_BUNDLE_MAX_FILES",
		defaultBundleMaxFiles,
	)
	bundleMaxBytes = int64(configuredPositiveInt(
		"REANA_SPEC_BUNDLE_MAX_BYTES",
		defaultBundleMaxBytes,
	))
	bundleMaxPath = configuredPositiveInt(
		"REANA_SPEC_BUNDLE_MAX_PATH_BYTES",
		defaultBundleMaxPath,
	)
)

func configuredPositiveInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

type specForBundle struct {
	Workflow workflowForBundle `yaml:"workflow"`
	Inputs   struct {
		Files       []string               `yaml:"files"`
		Directories []string               `yaml:"directories"`
		Parameters  map[string]interface{} `yaml:"parameters"`
		Options     yaml.Node              `yaml:"options"`
	} `yaml:"inputs"`
}

type workflowForBundle struct {
	Type        string   `yaml:"type"`
	File        string   `yaml:"file"`
	Files       []string `yaml:"files"`
	Directories []string `yaml:"directories"`
	Parameters  struct {
		File string `yaml:"file"`
	} `yaml:"parameters"`
	hasExplicitScope bool `yaml:"-"`
}

func (workflow *workflowForBundle) UnmarshalYAML(node *yaml.Node) error {
	type decodedWorkflow workflowForBundle
	var decoded decodedWorkflow
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	var mapping map[string]interface{}
	if err := node.Decode(&mapping); err != nil {
		return err
	}
	*workflow = workflowForBundle(decoded)
	_, hasFiles := mapping["files"]
	_, hasDirectories := mapping["directories"]
	workflow.hasExplicitScope = hasFiles || hasDirectories
	return nil
}

type scopeBudget struct {
	files                map[string]struct{}
	directories          map[string]struct{}
	traversedDirectories map[string]struct{}
}

func newScopeBudget() *scopeBudget {
	return &scopeBudget{
		files:                make(map[string]struct{}),
		directories:          make(map[string]struct{}),
		traversedDirectories: make(map[string]struct{}),
	}
}

func (budget *scopeBudget) addDirectory(relativeDirectory string) error {
	components := strings.Split(relativeDirectory, "/")
	if len(components) > bundleMaxDepth {
		return fmt.Errorf(
			"path exceeds the maximum depth of %d components: %s",
			bundleMaxDepth,
			relativeDirectory,
		)
	}
	for index := 1; index <= len(components); index++ {
		directory := strings.Join(components[:index], "/")
		if _, present := budget.directories[directory]; present {
			continue
		}
		if len(budget.directories) >= bundleMaxDirectories {
			return fmt.Errorf(
				"specification scope has too many directories (maximum is %d)",
				bundleMaxDirectories,
			)
		}
		budget.directories[directory] = struct{}{}
	}
	return nil
}

func (budget *scopeBudget) addFile(relativePath string) error {
	components := strings.Split(relativePath, "/")
	if len(components) > bundleMaxDepth {
		return fmt.Errorf(
			"path exceeds the maximum depth of %d components: %s",
			bundleMaxDepth,
			relativePath,
		)
	}
	if parent := path.Dir(relativePath); parent != "." {
		if err := budget.addDirectory(parent); err != nil {
			return err
		}
	}
	if _, present := budget.files[relativePath]; present {
		return nil
	}
	if len(budget.files) >= bundleMaxFiles {
		return fmt.Errorf(
			"specification scope has too many files (maximum is %d)",
			bundleMaxFiles,
		)
	}
	budget.files[relativePath] = struct{}{}
	return nil
}

func normalizeDeclaredPath(value, field string) (string, error) {
	if len([]byte(value)) > bundleMaxPath {
		return "", fmt.Errorf(
			"path in %s exceeds %d encoded bytes",
			field,
			bundleMaxPath,
		)
	}
	if value == "" || strings.ContainsRune(value, '\x00') ||
		strings.Contains(value, "\\") || strings.HasPrefix(value, "/") ||
		(len(value) >= 2 && value[1] == ':') {
		return "", fmt.Errorf("unsafe path in %s: %s", field, value)
	}
	normalized := path.Clean(value)
	if normalized == "." || normalized == ".." ||
		normalized != strings.TrimSuffix(value, "/") {
		return "", fmt.Errorf("unsafe path in %s: %s", field, value)
	}
	for _, component := range strings.Split(normalized, "/") {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("unsafe path in %s: %s", field, value)
		}
	}
	if len(strings.Split(normalized, "/")) > bundleMaxDepth {
		return "", fmt.Errorf(
			"path in %s exceeds the maximum depth of %d components",
			field,
			bundleMaxDepth,
		)
	}
	return normalized, nil
}

func ensureRegularPath(baseDir, relativePath, field string) (string, error) {
	current := baseDir
	components := strings.Split(relativePath, "/")
	for index, component := range components {
		current = filepath.Join(current, filepath.FromSlash(component))
		info, err := os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf(
				"declared path does not exist in %s: %s",
				field,
				relativePath,
			)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf(
				"symbolic links are not allowed in %s: %s",
				field,
				relativePath,
			)
		}
		if index < len(components)-1 && !info.IsDir() {
			return "", fmt.Errorf(
				"declared path is not a directory in %s: %s",
				field,
				relativePath,
			)
		}
		if index == len(components)-1 && !info.Mode().IsRegular() {
			return "", fmt.Errorf(
				"declared path is not a regular file in %s: %s",
				field,
				relativePath,
			)
		}
	}
	return current, nil
}

func addFile(
	members map[string]string,
	budget *scopeBudget,
	baseDir, declaration, field string,
	allowCanonicalInput bool,
) error {
	relativePath, err := normalizeDeclaredPath(declaration, field)
	if err != nil {
		return err
	}
	if relativePath == canonicalSpecification {
		if allowCanonicalInput && field == "inputs.files" {
			return nil
		}
		return fmt.Errorf(
			"%s cannot declare the reserved path %s",
			field,
			canonicalSpecification,
		)
	}
	absolutePath, err := ensureRegularPath(baseDir, relativePath, field)
	if err != nil {
		return err
	}
	if previous, ok := members[relativePath]; ok && previous != absolutePath {
		return fmt.Errorf(
			"conflicting declarations resolve to %s",
			relativePath,
		)
	}
	if err := budget.addFile(relativePath); err != nil {
		return err
	}
	members[relativePath] = absolutePath
	return nil
}

func addDirectory(
	members map[string]string,
	budget *scopeBudget,
	baseDir, declaration, field string,
) error {
	relativeDirectory, err := normalizeDeclaredPath(declaration, field)
	if err != nil {
		return err
	}
	if _, traversed := budget.traversedDirectories[relativeDirectory]; traversed {
		return nil
	}
	absoluteDirectory := filepath.Join(
		baseDir,
		filepath.FromSlash(relativeDirectory),
	)
	info, err := os.Lstat(absoluteDirectory)
	if err != nil {
		return fmt.Errorf(
			"declared directory does not exist in %s: %s",
			field,
			relativeDirectory,
		)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf(
			"declared path is not a regular directory in %s: %s",
			field,
			relativeDirectory,
		)
	}
	if err := budget.addDirectory(relativeDirectory); err != nil {
		return err
	}
	hasRegularFile := false
	pending := []string{relativeDirectory}
	for len(pending) > 0 {
		currentRelative := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		currentPath := filepath.Join(
			baseDir,
			filepath.FromSlash(currentRelative),
		)
		directory, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		for {
			entries, readErr := directory.ReadDir(1)
			if readErr != nil && readErr != io.EOF {
				_ = directory.Close()
				return readErr
			}
			if len(entries) == 0 {
				break
			}
			entry := entries[0]
			relative := path.Join(currentRelative, entry.Name())
			entryPath := filepath.Join(currentPath, entry.Name())
			info, err := os.Lstat(entryPath)
			if err != nil {
				_ = directory.Close()
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				_ = directory.Close()
				return fmt.Errorf(
					"symbolic links are not allowed in %s: %s",
					field,
					relative,
				)
			}
			if info.IsDir() {
				if err := budget.addDirectory(relative); err != nil {
					_ = directory.Close()
					return err
				}
				pending = append(pending, relative)
			} else if info.Mode().IsRegular() {
				hasRegularFile = true
				if err := addFile(members, budget, baseDir, relative, field, false); err != nil {
					_ = directory.Close()
					return err
				}
			} else {
				_ = directory.Close()
				return fmt.Errorf(
					"only regular files are allowed in %s: %s",
					field,
					relative,
				)
			}
			if readErr == io.EOF {
				break
			}
		}
		if err := directory.Close(); err != nil {
			return err
		}
	}
	if !hasRegularFile {
		return fmt.Errorf(
			"declared directory contains no regular files in %s: %s",
			field,
			relativeDirectory,
		)
	}
	budget.traversedDirectories[relativeDirectory] = struct{}{}
	return nil
}

// readSpecification returns a securely opened, bounded specification and path.
func readSpecification(reanaFile string) ([]byte, string, error) {
	absoluteSpecification, err := filepath.Abs(reanaFile)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Lstat(absoluteSpecification)
	if err != nil {
		return nil, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf(
			"the selected REANA specification must be a regular file",
		)
	}
	file, err := openRegularFileBeneath(
		filepath.Dir(absoluteSpecification),
		filepath.Base(absoluteSpecification),
		"REANA specification",
	)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = file.Close() }()
	metadata, err := file.Stat()
	if err != nil {
		return nil, "", err
	}
	if metadata.Size() > bundleMaxBytes {
		return nil, "", fmt.Errorf(
			"REANA specification is too large (maximum is %d bytes)",
			bundleMaxBytes,
		)
	}
	contents, err := io.ReadAll(io.LimitReader(file, bundleMaxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(contents)) > bundleMaxBytes {
		return nil, "", fmt.Errorf(
			"REANA specification is too large (maximum is %d bytes)",
			bundleMaxBytes,
		)
	}
	return contents, absoluteSpecification, nil
}

// ReadSpecification securely reads one specification within the bundle limit.
func ReadSpecification(reanaFile string) ([]byte, error) {
	contents, _, err := readSpecification(reanaFile)
	return contents, err
}

// SpecificationFile is a securely opened replacement with a stable absolute
// name for go-swagger's multipart transport.
type SpecificationFile struct {
	file      *os.File
	name      string
	limit     int64
	remaining int64
	tooLarge  bool
}

// Name returns the validated absolute path used for multipart metadata.
func (file *SpecificationFile) Name() string {
	return file.name
}

// Read streams at most the configured specification limit and reports files
// that grow beyond it before any excess byte is returned to the transport.
func (file *SpecificationFile) Read(contents []byte) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	if file.tooLarge {
		return 0, fmt.Errorf(
			"REANA specification is too large (maximum is %d bytes)",
			file.limit,
		)
	}
	if file.remaining > 0 {
		readLimit := int64(len(contents))
		if readLimit > file.remaining {
			readLimit = file.remaining
		}
		bytesRead, err := file.file.Read(contents[:readLimit])
		file.remaining -= int64(bytesRead)
		return bytesRead, err
	}

	var overflow [1]byte
	bytesRead, err := file.file.Read(overflow[:])
	if bytesRead > 0 {
		file.tooLarge = true
		return 0, fmt.Errorf(
			"REANA specification is too large (maximum is %d bytes)",
			file.limit,
		)
	}
	return 0, err
}

// Close closes the validated specification descriptor.
func (file *SpecificationFile) Close() error {
	return file.file.Close()
}

// OpenSpecification securely opens one bounded specification for streaming.
// The caller owns the returned descriptor.
func OpenSpecification(reanaFile string) (*SpecificationFile, error) {
	absoluteSpecification, err := filepath.Abs(reanaFile)
	if err != nil {
		return nil, err
	}
	file, err := openRegularFileBeneath(
		filepath.Dir(absoluteSpecification),
		filepath.Base(absoluteSpecification),
		"REANA specification",
	)
	if err != nil {
		return nil, err
	}
	metadata, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if metadata.Size() > bundleMaxBytes {
		_ = file.Close()
		return nil, fmt.Errorf(
			"REANA specification is too large (maximum is %d bytes)",
			bundleMaxBytes,
		)
	}
	return &SpecificationFile{
		file:      file,
		name:      absoluteSpecification,
		limit:     bundleMaxBytes,
		remaining: bundleMaxBytes,
	}, nil
}

// Gather returns the canonical specification plus explicitly declared workflow
// definition and parameter files. Input datasets are uploaded separately.
func Gather(reanaFile string) (map[string]string, error) {
	contents, absoluteSpecification, err := readSpecification(reanaFile)
	if err != nil {
		return nil, err
	}
	members := map[string]string{canonicalSpecification: absoluteSpecification}
	budget := newScopeBudget()
	if err := budget.addFile(canonicalSpecification); err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(contents, &document); err != nil {
		return members, nil
	}
	if len(document.Content) != 1 ||
		document.Content[0].Kind != yaml.MappingNode ||
		len(document.Content[0].Content) == 0 {
		return members, nil
	}
	var specification specForBundle
	if err := yaml.Unmarshal(contents, &specification); err != nil {
		return members, nil
	}

	baseDir := filepath.Dir(absoluteSpecification)
	if specification.Workflow.File != "" {
		workflowFile := specification.Workflow.File
		if specification.Workflow.Type == "cwl" {
			workflowFile, _, _ = strings.Cut(workflowFile, "#")
			if workflowFile == "" {
				return nil, fmt.Errorf(
					"workflow.file must identify a local CWL document before its fragment",
				)
			}
		}
		if err := addFile(members, budget, baseDir, workflowFile, "workflow.file", false); err != nil {
			return nil, err
		}
	}
	if len(
		specification.Workflow.Files,
	)+len(
		specification.Workflow.Directories,
	) >
		bundleMaxFiles {
		return nil, fmt.Errorf(
			"specification scope has too many declared paths in workflow (maximum is %d)",
			bundleMaxFiles,
		)
	}
	for _, filename := range specification.Workflow.Files {
		if err := addFile(members, budget, baseDir, filename, "workflow.files", false); err != nil {
			return nil, err
		}
	}
	for _, directory := range specification.Workflow.Directories {
		if err := addDirectory(members, budget, baseDir, directory, "workflow.directories"); err != nil {
			return nil, err
		}
	}
	if specification.Workflow.File != "" &&
		!specification.Workflow.hasExplicitScope {
		if len(
			specification.Inputs.Files,
		)+len(
			specification.Inputs.Directories,
		) >
			bundleMaxFiles {
			return nil, fmt.Errorf(
				"specification scope has too many declared paths in inputs (maximum is %d)",
				bundleMaxFiles,
			)
		}
		for _, filename := range specification.Inputs.Files {
			if err := addFile(
				members,
				budget,
				baseDir,
				filename,
				"inputs.files",
				filepath.Base(absoluteSpecification) == canonicalSpecification,
			); err != nil {
				return nil, err
			}
		}
		for _, directory := range specification.Inputs.Directories {
			if err := addDirectory(
				members,
				budget,
				baseDir,
				directory,
				"inputs.directories",
			); err != nil {
				return nil, err
			}
		}
	}

	parameterFile := specification.Workflow.Parameters.File
	legacyFile := ""
	if specification.Workflow.Type == "cwl" ||
		specification.Workflow.Type == "snakemake" {
		if value, present := specification.Inputs.Parameters["input"]; present {
			var ok bool
			legacyFile, ok = value.(string)
			if !ok {
				return map[string]string{
					canonicalSpecification: absoluteSpecification,
				}, nil
			}
		}
	}
	if parameterFile != "" && specification.Workflow.Type != "cwl" &&
		specification.Workflow.Type != "snakemake" {
		return nil, fmt.Errorf(
			"workflow.parameters.file is supported only for CWL and Snakemake workflows",
		)
	}
	if parameterFile != "" && legacyFile != "" {
		return nil, fmt.Errorf(
			"use either workflow.parameters.file or inputs.parameters.input, not both",
		)
	}
	if parameterFile != "" {
		if err := addFile(members, budget, baseDir, parameterFile, "workflow.parameters.file", false); err != nil {
			return nil, err
		}
	} else if legacyFile != "" {
		if err := addFile(members, budget, baseDir, legacyFile, "inputs.parameters.input", false); err != nil {
			return nil, err
		}
	}
	if specification.Workflow.Type == "yadage" &&
		specification.Inputs.Options.Kind != 0 {
		var value interface{}
		if err := specification.Inputs.Options.Decode(&value); err != nil {
			return nil, fmt.Errorf("invalid inputs.options: %w", err)
		}
		options, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("inputs.options must be a mapping")
		}
		if value, present := options["initfiles"]; present {
			initfiles, ok := value.([]interface{})
			if !ok {
				return nil, fmt.Errorf(
					"inputs.options.initfiles must be a list",
				)
			}
			for _, value := range initfiles {
				filename, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf(
						"inputs.options.initfiles entries must be strings",
					)
				}
				if err := addFile(members, budget, baseDir, filename, "inputs.options.initfiles", false); err != nil {
					return nil, err
				}
			}
		}
	}
	return members, nil
}

type temporaryArchive struct {
	file *os.File
	path string
}

func (archive *temporaryArchive) Read(buffer []byte) (int, error) {
	return archive.file.Read(buffer)
}

func (archive *temporaryArchive) Name() string {
	return "validation-bundle.zip"
}

func (archive *temporaryArchive) Close() error {
	if archive.file == nil {
		return nil
	}
	closeErr := archive.file.Close()
	archive.file = nil
	removeErr := os.Remove(archive.path)
	if closeErr != nil {
		return closeErr
	}
	if os.IsNotExist(removeErr) {
		return nil
	}
	return removeErr
}

// Archive writes a deterministic, uncompressed ZIP snapshot to a temporary
// file suitable for the generated client's multipart file parameter.
func Archive(members map[string]string) (*temporaryArchive, error) {
	if len(members) > bundleMaxFiles {
		return nil, fmt.Errorf(
			"specification bundle has too many files (maximum is %d)",
			bundleMaxFiles,
		)
	}
	specificationPath, present := members[canonicalSpecification]
	if !present {
		return nil, fmt.Errorf(
			"specification bundle is missing canonical reana.yaml",
		)
	}
	absoluteSpecification, err := filepath.Abs(specificationPath)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(absoluteSpecification)

	names := make([]string, 0, len(members))
	budget := newScopeBudget()
	for name := range members {
		if err := budget.addFile(name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	sort.Strings(names)

	file, err := os.CreateTemp("", "reana-validation-*.zip")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}

	writer := zip.NewWriter(file)
	var totalBytes int64
	buffer := make([]byte, 1024*1024)
	for _, name := range names {
		archiveName, err := normalizeDeclaredPath(name, "specification bundle")
		if err != nil {
			_ = writer.Close()
			cleanup()
			return nil, err
		}
		sourcePath, err := filepath.Abs(members[name])
		if err != nil {
			_ = writer.Close()
			cleanup()
			return nil, err
		}
		sourceRelativePath, err := filepath.Rel(baseDir, sourcePath)
		if err != nil {
			_ = writer.Close()
			cleanup()
			return nil, err
		}
		sourceRelativePath, err = normalizeDeclaredPath(
			filepath.ToSlash(sourceRelativePath),
			"specification bundle",
		)
		if err != nil {
			_ = writer.Close()
			cleanup()
			return nil, err
		}
		header := &zip.FileHeader{Name: archiveName, Method: zip.Store}
		header.SetMode(0o600)
		header.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		target, createErr := writer.CreateHeader(header)
		if createErr != nil {
			_ = writer.Close()
			cleanup()
			return nil, createErr
		}
		source, openErr := openRegularFileBeneath(
			baseDir,
			sourceRelativePath,
			"specification bundle",
		)
		if openErr != nil {
			_ = writer.Close()
			cleanup()
			return nil, openErr
		}
		var copyErr error
		for {
			readBytes, readErr := source.Read(buffer)
			if readBytes > 0 {
				totalBytes += int64(readBytes)
				if totalBytes > bundleMaxBytes {
					copyErr = fmt.Errorf(
						"specification bundle is too large (maximum is %d bytes)",
						bundleMaxBytes,
					)
					break
				}
				if _, writeErr := target.Write(buffer[:readBytes]); writeErr != nil {
					copyErr = writeErr
					break
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				copyErr = readErr
				break
			}
		}
		closeErr := source.Close()
		if copyErr != nil {
			_ = writer.Close()
			cleanup()
			return nil, copyErr
		}
		if closeErr != nil {
			_ = writer.Close()
			cleanup()
			return nil, closeErr
		}
	}
	if err := writer.Close(); err != nil {
		cleanup()
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, err
	}
	return &temporaryArchive{file: file, path: file.Name()}, nil
}
