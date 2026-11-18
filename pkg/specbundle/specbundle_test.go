/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package specbundle

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, name, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGatherExplicitScope(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workflow", "Snakefile"), "")
	writeFile(t, filepath.Join(root, "workflow", "undeclared.smk"), "")
	writeFile(t, filepath.Join(root, "rules", "common.smk"), "")
	writeFile(t, filepath.Join(root, "extra.yaml"), "value: 1")
	writeFile(t, filepath.Join(root, "params.yaml"), "answer: 42")
	writeFile(t, filepath.Join(root, "dataset.csv"), "data")
	specification := filepath.Join(root, "selected.yaml")
	writeFile(t, specification, `
workflow:
  type: snakemake
  file: workflow/Snakefile
  files: [extra.yaml]
  directories: [rules]
  parameters:
    file: params.yaml
inputs:
  files: [dataset.csv]
`)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"reana.yaml",
		"workflow/Snakefile",
		"rules/common.smk",
		"extra.yaml",
		"params.yaml",
	}
	for _, member := range expected {
		if _, ok := members[member]; !ok {
			t.Fatalf("expected %q in %#v", member, members)
		}
	}
	if _, ok := members["workflow/undeclared.smk"]; ok {
		t.Fatal("undeclared workflow sibling was bundled")
	}
	if _, ok := members["dataset.csv"]; ok {
		t.Fatal("input dataset was bundled for validation")
	}
}

func TestGatherLegacyExternalWorkflowScope(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workflow", "main.cwl"), "class: Workflow")
	writeFile(
		t,
		filepath.Join(root, "workflow", "step.cwl"),
		"class: CommandLineTool",
	)
	writeFile(t, filepath.Join(root, "data.txt"), "data")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
inputs:
  files: [data.txt]
  directories: [workflow]
workflow:
  type: cwl
  file: workflow/main.cwl
`)

	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range []string{
		"reana.yaml",
		"data.txt",
		"workflow/main.cwl",
		"workflow/step.cwl",
	} {
		if _, present := members[member]; !present {
			t.Fatalf("expected %q in %#v", member, members)
		}
	}
}

func TestGatherExplicitEmptyScopeDisablesLegacyFallback(t *testing.T) {
	for _, explicit := range []string{"files: []", "directories: []"} {
		t.Run(explicit, func(t *testing.T) {
			root := t.TempDir()
			writeFile(
				t,
				filepath.Join(root, "workflow", "main.cwl"),
				"class: Workflow",
			)
			writeFile(
				t,
				filepath.Join(root, "workflow", "step.cwl"),
				"class: CommandLineTool",
			)
			specification := filepath.Join(root, "reana.yaml")
			writeFile(t, specification, `
inputs:
  directories: [workflow]
workflow:
  type: cwl
  file: workflow/main.cwl
  `+explicit+`
`)

			members, err := Gather(specification)
			if err != nil {
				t.Fatal(err)
			}
			if _, present := members["workflow/step.cwl"]; present {
				t.Fatal(
					"legacy input directory entered explicit workflow scope",
				)
			}
		})
	}
}

func TestGatherMergedExplicitScopeDisablesLegacyFallback(t *testing.T) {
	for _, scope := range []string{"files: []", "directories: []"} {
		t.Run(scope, func(t *testing.T) {
			root := t.TempDir()
			writeFile(
				t,
				filepath.Join(root, "workflow", "main.cwl"),
				"class: Workflow",
			)
			writeFile(t, filepath.Join(root, "data.txt"), "runtime input")
			specification := filepath.Join(root, "reana.yaml")
			writeFile(t, specification, `
inputs:
  files: [data.txt]
workflow:
  <<: &scope
    `+scope+`
  type: cwl
  file: workflow/main.cwl
`)

			members, err := Gather(specification)
			if err != nil {
				t.Fatal(err)
			}
			if _, present := members["data.txt"]; present {
				t.Fatal("legacy input entered merge-inherited workflow scope")
			}
		})
	}
}

func TestGatherExplicitNullScopeDisablesLegacyFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workflow", "main.cwl"), "class: Workflow")
	writeFile(t, filepath.Join(root, "data.txt"), "runtime input")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
inputs:
  files: [data.txt]
workflow:
  type: cwl
  file: workflow/main.cwl
  files: null
`)

	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := members["data.txt"]; present {
		t.Fatal("legacy input entered explicitly null workflow scope")
	}
}

func TestGatherInlineSerialDoesNotUseLegacyFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "large.dat"), "runtime input")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
inputs:
  files: [large.dat]
workflow:
  type: serial
  specification: {steps: []}
`)

	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members["reana.yaml"] == "" {
		t.Fatalf("unexpected serial validation scope: %#v", members)
	}
}

func TestGatherAllowsOverlappingDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workflow", "main.cwl"), "class: Workflow")
	writeFile(
		t,
		filepath.Join(root, "workflow", "steps", "step.cwl"),
		"class: CommandLineTool",
	)
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
workflow:
  type: cwl
  file: workflow/main.cwl
  directories: [workflow, workflow/steps]
`)

	if _, err := Gather(specification); err != nil {
		t.Fatalf("overlapping non-empty directories were rejected: %v", err)
	}
}

func TestGatherAllowsRepeatedDirectoryRoots(t *testing.T) {
	root := t.TempDir()
	writeFile(
		t,
		filepath.Join(root, "workflow", "step.cwl"),
		"class: CommandLineTool",
	)
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
workflow:
  type: serial
  specification: {steps: []}
  directories: [workflow, workflow]
`)

	members, err := Gather(specification)
	if err != nil {
		t.Fatalf("repeated directory root was rejected: %v", err)
	}
	if len(members) != 2 || members["workflow/step.cwl"] == "" {
		t.Fatalf("unexpected repeated-directory members: %#v", members)
	}
}

func TestGatherBoundsCombinedDeclarationsBeforeTraversal(t *testing.T) {
	previous := bundleMaxFiles
	bundleMaxFiles = 2
	defer func() { bundleMaxFiles = previous }()
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
workflow:
  type: serial
  specification: {steps: []}
  files: [missing, missing]
  directories: [missing]
`)

	_, err := Gather(specification)
	if err == nil || !strings.Contains(err.Error(), "too many declared paths") {
		t.Fatalf("unexpected declaration-limit error: %v", err)
	}
	if strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("oversized declarations reached filesystem traversal: %v", err)
	}
}

func TestGatherBoundsLegacyInputDeclarationsBeforeTraversal(t *testing.T) {
	previous := bundleMaxFiles
	bundleMaxFiles = 2
	defer func() { bundleMaxFiles = previous }()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.cwl"), "class: Workflow")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
inputs:
  files: [missing, missing]
  directories: [missing]
workflow:
  type: cwl
  file: main.cwl
`)

	_, err := Gather(specification)
	if err == nil ||
		!strings.Contains(err.Error(), "too many declared paths in inputs") {
		t.Fatalf("unexpected legacy declaration-limit error: %v", err)
	}
	if strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("oversized legacy declarations reached traversal: %v", err)
	}
}

func TestGatherLegacyParameters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Snakefile"), "")
	writeFile(t, filepath.Join(root, "params.yaml"), "answer: 42")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
inputs:
  parameters:
    input: params.yaml
workflow:
  type: snakemake
  file: Snakefile
`)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := members["params.yaml"]; !ok {
		t.Fatal("legacy parameter file is missing")
	}
}

func TestGatherRejectsUnsafeAndSymlinkPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Snakefile"), "")
	writeFile(t, filepath.Join(root, "target.smk"), "")
	if err := os.Symlink("target.smk", filepath.Join(root, "linked.smk")); err != nil {
		t.Fatal(err)
	}
	for name, declaration := range map[string]string{
		"escape":   "../secret",
		"absolute": "/etc/passwd",
		"symlink":  "linked.smk",
	} {
		t.Run(name, func(t *testing.T) {
			specification := filepath.Join(root, "reana-"+name+".yaml")
			writeFile(
				t,
				specification,
				"workflow:\n  type: snakemake\n  file: Snakefile\n  files: ["+declaration+"]\n",
			)
			if _, err := Gather(specification); err == nil {
				t.Fatal("expected unsafe declaration to fail")
			}
		})
	}
}

func TestNormalizeDeclaredPath(t *testing.T) {
	for _, value := range []string{
		"",
		".",
		"../secret",
		"./workflow.cwl",
		"rules/../secret",
		"rules//common.smk",
		`rules\common.smk`,
		"C:/workflow.cwl",
		"/workflow.cwl",
		"workflow\x00.cwl",
	} {
		t.Run(strings.ReplaceAll(value, "/", "_"), func(t *testing.T) {
			if _, err := normalizeDeclaredPath(value, "test field"); err == nil {
				t.Fatalf("expected %q to be rejected", value)
			}
		})
	}
	if got, err := normalizeDeclaredPath(
		"rules/common.smk", "test field",
	); err != nil || got != "rules/common.smk" {
		t.Fatalf("unexpected normalized path %q, %v", got, err)
	}
}

func TestConfiguredPositiveInt(t *testing.T) {
	t.Setenv("REANA_TEST_LIMIT", "42")
	if got := configuredPositiveInt("REANA_TEST_LIMIT", 10); got != 42 {
		t.Errorf("expected configured limit, got %d", got)
	}
	for _, value := range []string{"", "invalid", "0", "-1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("REANA_TEST_LIMIT", value)
			if got := configuredPositiveInt("REANA_TEST_LIMIT", 10); got != 10 {
				t.Errorf("expected fallback limit, got %d", got)
			}
		})
	}
}

func TestGatherRejectsInvalidDeclarations(t *testing.T) {
	for name, contents := range map[string]string{
		"empty-cwl-fragment": `
workflow:
  type: cwl
  file: "#tool"
`,
		"unsupported-parameter-file": `
workflow:
  type: serial
  parameters:
    file: params.yaml
`,
		"conflicting-parameter-files": `
inputs:
  parameters:
    input: legacy.yaml
workflow:
  type: snakemake
  parameters:
    file: params.yaml
`,
		"reserved-file": `
workflow:
  type: serial
  files: [reana.yaml]
`,
		"missing-file": `
workflow:
  type: serial
  files: [missing.yaml]
`,
	} {
		t.Run(name, func(t *testing.T) {
			specification := filepath.Join(t.TempDir(), "reana.yaml")
			writeFile(t, specification, contents)
			if _, err := Gather(specification); err == nil {
				t.Fatal("expected invalid declaration to fail")
			}
		})
	}
}

func TestGatherDeduplicatesLegacyCanonicalInput(t *testing.T) {
	directory := t.TempDir()
	specification := filepath.Join(directory, canonicalSpecification)
	workflow := filepath.Join(directory, "main.cwl")
	writeFile(t, workflow, "class: Workflow")
	writeFile(t, specification, `
inputs:
  files: [reana.yaml]
workflow:
  type: cwl
  file: main.cwl
`)

	members, err := Gather(specification)
	if err != nil {
		t.Fatalf("gather legacy specification: %v", err)
	}
	if len(members) != 2 ||
		members[canonicalSpecification] != specification ||
		members["main.cwl"] != workflow {
		t.Fatalf("unexpected members: %#v", members)
	}
}

func TestGatherRejectsSiblingCanonicalInput(t *testing.T) {
	directory := t.TempDir()
	specification := filepath.Join(directory, "selected.yaml")
	writeFile(
		t,
		filepath.Join(directory, canonicalSpecification),
		"workflow: {type: serial}\n",
	)
	writeFile(t, filepath.Join(directory, "main.cwl"), "class: Workflow")
	writeFile(t, specification, `
inputs:
  files: [reana.yaml]
workflow:
  type: cwl
  file: main.cwl
`)

	if _, err := Gather(specification); err == nil {
		t.Fatal("expected canonical member conflict")
	}
}

func TestGatherRejectsInvalidFilesystemTypes(t *testing.T) {
	t.Run("selected specification directory", func(t *testing.T) {
		if _, err := Gather(t.TempDir()); err == nil ||
			!strings.Contains(err.Error(), "regular file") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("selected specification symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "selected.yaml")
		writeFile(t, target, "workflow: {type: serial}\n")
		link := filepath.Join(root, "reana.yaml")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := Gather(link); err == nil ||
			!strings.Contains(err.Error(), "regular file") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("file ancestor is not a directory", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "rules"), "not a directory")
		specification := filepath.Join(root, "reana.yaml")
		writeFile(t, specification, `
workflow:
  type: serial
  files: [rules/common.smk]
`)
		if _, err := Gather(specification); err == nil ||
			!strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("directory contains symlink", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "outside.smk"), "outside")
		if err := os.Mkdir(filepath.Join(root, "rules"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(
			"../outside.smk",
			filepath.Join(root, "rules", "linked.smk"),
		); err != nil {
			t.Fatal(err)
		}
		specification := filepath.Join(root, "reana.yaml")
		writeFile(t, specification, `
workflow:
  type: serial
  directories: [rules]
`)
		if _, err := Gather(specification); err == nil ||
			!strings.Contains(err.Error(), "symbolic links") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGatherFallsBackToCanonicalForInvalidSpecificationShapes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Snakefile"), "")
	for name, contents := range map[string]string{
		"invalid-yaml": "[unterminated\n",
		"scalar":       "not-a-mapping\n",
		"empty":        "",
		"legacy-parameter-type": `
workflow:
  type: snakemake
  file: Snakefile
inputs:
  parameters:
    input: [not, a, path]
`,
	} {
		t.Run(name, func(t *testing.T) {
			specification := filepath.Join(root, name+".yaml")
			writeFile(t, specification, contents)
			members, err := Gather(specification)
			if err != nil {
				t.Fatal(err)
			}
			if len(members) != 1 ||
				members[canonicalSpecification] != specification {
				t.Fatalf("expected canonical-only fallback, got %#v", members)
			}
		})
	}
}

func TestGatherRejectsEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, `
workflow:
  type: serial
  directories: [empty]
  specification:
    steps: []
`)
	if _, err := Gather(specification); err == nil ||
		!strings.Contains(err.Error(), "contains no regular files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGatherCWLFragmentUsesBaseDocument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tools.cwl"), "$graph: []")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(
		t,
		specification,
		"workflow:\n  type: cwl\n  file: tools.cwl#selected-tool\n",
	)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := members["tools.cwl"]; !ok {
		t.Fatalf("CWL base document missing from %#v", members)
	}
	if _, ok := members["tools.cwl#selected-tool"]; ok {
		t.Fatal("CWL fragment leaked into archive path")
	}
}

func TestArchiveBuildsSingleStoredZip(t *testing.T) {
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(
		t,
		specification,
		"workflow:\n  type: serial\n  specification:\n    steps: []\n",
	)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}

	archive, err := Archive(members)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := archive.path
	defer func() {
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if archive.Name() != "validation-bundle.zip" {
		t.Fatalf("unexpected multipart filename %q", archive.Name())
	}
	info, err := archive.file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(archive.file, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 || reader.File[0].Name != "reana.yaml" {
		t.Fatalf("unexpected archive: %#v", reader.File)
	}
	if reader.File[0].Method != zip.Store {
		t.Fatal("bundle entry is compressed")
	}
	if archivePath == "" {
		t.Fatal("temporary archive path is empty")
	}
}

func TestArchivePreservesNoncanonicalSelectedSpecification(t *testing.T) {
	root := t.TempDir()
	specification := filepath.Join(root, "selected.yaml")
	contents := "workflow:\n  type: serial\n  specification:\n    steps: []\n"
	writeFile(t, specification, contents)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := Archive(members)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	info, err := archive.file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(archive.file, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	source, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	archived, err := io.ReadAll(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if reader.File[0].Name != canonicalSpecification ||
		string(archived) != contents {
		t.Fatalf("selected specification was not archived canonically")
	}
}

func TestArchiveRejectsSwappedSourceAncestor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "defs", "Snakefile"), "ORIGINAL")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(
		t,
		specification,
		"workflow:\n  type: snakemake\n  file: defs/Snakefile\n",
	)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "Snakefile"), "OUTSIDE")
	if err := os.Rename(filepath.Join(root, "defs"), filepath.Join(root, "defs-original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "defs")); err != nil {
		t.Fatal(err)
	}
	if _, err := Archive(members); err == nil ||
		!strings.Contains(err.Error(), "securely open") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchiveAcceptsSymlinkedTrustedBase(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Snakefile"), "ORIGINAL")
	writeFile(
		t,
		filepath.Join(root, "reana.yaml"),
		"workflow:\n  type: snakemake\n  file: Snakefile\n",
	)
	alias := filepath.Join(t.TempDir(), "trusted-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	members, err := Gather(filepath.Join(alias, "reana.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	archive, err := Archive(members)
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScopeBudgetBoundsDirectoriesAndDepth(t *testing.T) {
	budget := newScopeBudget()
	for index := 0; index < bundleMaxDirectories; index++ {
		if err := budget.addDirectory(fmt.Sprintf("directory-%d", index)); err != nil {
			t.Fatal(err)
		}
	}
	if err := budget.addDirectory("one-too-many"); err == nil ||
		!strings.Contains(err.Error(), "too many directories") {
		t.Fatalf("unexpected directory-limit error: %v", err)
	}

	components := make([]string, bundleMaxDepth+1)
	for index := range components {
		components[index] = "nested"
	}
	if err := newScopeBudget().addFile(strings.Join(components, "/")); err == nil ||
		!strings.Contains(err.Error(), "maximum depth") {
		t.Fatalf("unexpected depth-limit error: %v", err)
	}
}

func TestArchiveRejectsSwappedSourceFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "Snakefile")
	writeFile(t, source, "ORIGINAL")
	specification := filepath.Join(root, "reana.yaml")
	writeFile(
		t,
		specification,
		"workflow:\n  type: snakemake\n  file: Snakefile\n",
	)
	members, err := Gather(specification)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.smk")
	writeFile(t, outside, "OUTSIDE")
	if err := os.Rename(source, source+"-original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, source); err != nil {
		t.Fatal(err)
	}
	if _, err := Archive(members); err == nil ||
		!strings.Contains(err.Error(), "securely open") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchiveRejectsMissingCanonicalAndOutsideSource(t *testing.T) {
	if _, err := Archive(map[string]string{}); err == nil ||
		!strings.Contains(err.Error(), "missing canonical") {
		t.Fatalf("unexpected missing canonical error: %v", err)
	}

	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, "workflow: {type: serial}\n")
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	writeFile(t, outside, "outside")
	if _, err := Archive(map[string]string{
		canonicalSpecification: specification,
		"outside.yaml":         outside,
	}); err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("unexpected outside source error: %v", err)
	}
}

func TestTemporaryArchiveReadAndClose(t *testing.T) {
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, "workflow: {type: serial}\n")
	archive, err := Archive(map[string]string{
		canonicalSpecification: specification,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := archive.path
	if contents, err := io.ReadAll(archive); err != nil || len(contents) == 0 {
		t.Fatalf("could not read archive: %d bytes, %v", len(contents), err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary archive was not removed: %v", err)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("second close should be harmless: %v", err)
	}
}

func TestArchiveEnforcesFileCountBeforeWriting(t *testing.T) {
	previous := bundleMaxFiles
	bundleMaxFiles = 1
	defer func() { bundleMaxFiles = previous }()
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, "workflow: {type: serial}\n")
	extra := filepath.Join(root, "extra.yaml")
	writeFile(t, extra, "value: 1\n")
	if _, err := Archive(map[string]string{
		canonicalSpecification: specification,
		"extra.yaml":           extra,
	}); err == nil || !strings.Contains(err.Error(), "too many files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchiveEnforcesMemberPathLength(t *testing.T) {
	previous := bundleMaxPath
	bundleMaxPath = 10
	defer func() { bundleMaxPath = previous }()
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, "workflow: {type: serial}\n")
	extra := filepath.Join(root, "long-name.yaml")
	writeFile(t, extra, "value: 1\n")
	if _, err := Archive(map[string]string{
		canonicalSpecification: specification,
		"long-name.yaml":       extra,
	}); err == nil || !strings.Contains(err.Error(), "encoded bytes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchiveEnforcesExtractedBytesWhileWriting(t *testing.T) {
	previous := bundleMaxBytes
	bundleMaxBytes = 10
	defer func() { bundleMaxBytes = previous }()
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(t, specification, "workflow: {type: serial}\n")
	if _, err := Archive(map[string]string{
		canonicalSpecification: specification,
	}); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSpecificationReadsAreBounded(t *testing.T) {
	previous := bundleMaxBytes
	bundleMaxBytes = 8
	defer func() { bundleMaxBytes = previous }()
	path := filepath.Join(t.TempDir(), "reana.yaml")
	writeFile(t, path, "123456789")

	for name, operation := range map[string]func() error{
		"gather": func() error { _, err := Gather(path); return err },
		"read":   func() error { _, err := ReadSpecification(path); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := operation(); err == nil ||
				!strings.Contains(err.Error(), "too large") {
				t.Fatalf("unexpected bounded-read error: %v", err)
			}
		})
	}
}

func TestOpenSpecificationRejectsInitiallyOversizedFile(t *testing.T) {
	previous := bundleMaxBytes
	bundleMaxBytes = 8
	defer func() { bundleMaxBytes = previous }()
	path := filepath.Join(t.TempDir(), "reana.yaml")
	writeFile(t, path, "123456789")

	file, err := OpenSpecification(path)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("unexpected bounded-open error: %v", err)
	}
}

func TestOpenSpecificationDetectsGrowthWhileStreaming(t *testing.T) {
	previous := bundleMaxBytes
	bundleMaxBytes = 8
	defer func() { bundleMaxBytes = previous }()
	path := filepath.Join(t.TempDir(), "reana.yaml")
	writeFile(t, path, "1234")

	file, err := OpenSpecification(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	appender, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appender.WriteString("56789"); err != nil {
		_ = appender.Close()
		t.Fatal(err)
	}
	if err := appender.Close(); err != nil {
		t.Fatal(err)
	}

	contents, err := io.ReadAll(file)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("unexpected streaming error: %v", err)
	}
	if string(contents) != "12345678" {
		t.Fatalf("streamed beyond the byte limit: %q", contents)
	}
}

func TestOpenSpecificationStreamsExactFileAndClosesDescriptor(t *testing.T) {
	previous := bundleMaxBytes
	bundleMaxBytes = 8
	defer func() { bundleMaxBytes = previous }()
	path := filepath.Join(t.TempDir(), "reana.yaml")
	writeFile(t, path, "12345678")

	file, err := OpenSpecification(path)
	if err != nil {
		t.Fatal(err)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if file.Name() != absolutePath {
		t.Fatalf("unexpected specification name: %q", file.Name())
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "12345678" {
		t.Fatalf("unexpected specification contents: %q", contents)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Read(make([]byte, 1)); err == nil {
		t.Fatal("closed specification descriptor remained readable")
	}
}

func TestGatherYadageInitfiles(t *testing.T) {
	for name, scope := range map[string]struct{ workflow, inputs string }{
		"explicit-files":       {workflow: "  files: []\n"},
		"explicit-directories": {workflow: "  directories: []\n"},
		"implicit-scope":       {},
		"legacy-duplicate":     {inputs: "  files: [params.yaml]\n"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "workflow.yaml"), "stages: []\n")
			writeFile(t, filepath.Join(root, "params.yaml"), "events: 2000\n")
			specification := filepath.Join(root, "reana.yaml")
			writeFile(
				t,
				specification,
				"workflow:\n  type: yadage\n  file: workflow.yaml\n"+scope.workflow+
					"inputs:\n"+scope.inputs+"  options:\n    initfiles: [params.yaml]\n",
			)
			members, err := Gather(specification)
			if err != nil {
				t.Fatal(err)
			}
			if len(members) != 3 ||
				members["params.yaml"] != filepath.Join(root, "params.yaml") {
				t.Fatalf(
					"expected initfile in validation scope, got %#v",
					members,
				)
			}
		})
	}
}

func TestGatherRejectsInvalidYadageInitfiles(t *testing.T) {
	for name, options := range map[string]string{
		"options-list":     "[]",
		"options-null":     "null",
		"initfiles-string": "{initfiles: params.yaml}",
		"initfiles-null":   "{initfiles: null}",
		"entry-number":     "{initfiles: [42]}",
		"entry-mapping":    "{initfiles: [{file: params.yaml}]}",
		"missing":          "{initfiles: [missing.yaml]}",
		"absolute":         "{initfiles: [/params.yaml]}",
		"parent":           "{initfiles: [../params.yaml]}",
		"symlink":          "{initfiles: [link.yaml]}",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "workflow.yaml"), "stages: []\n")
			writeFile(t, filepath.Join(root, "params.yaml"), "events: 2000\n")
			if err := os.Symlink("params.yaml", filepath.Join(root, "link.yaml")); err != nil {
				t.Fatal(err)
			}
			specification := filepath.Join(root, "reana.yaml")
			writeFile(t, specification,
				"workflow:\n  type: yadage\n  file: workflow.yaml\n"+
					"inputs:\n  options: "+options+"\n")
			if _, err := Gather(specification); err == nil ||
				!strings.Contains(err.Error(), "inputs.options") {
				t.Fatalf("expected initfile declaration error, got %v", err)
			}
		})
	}
}

func TestGatherIgnoresInitfilesForOtherEngines(t *testing.T) {
	root := t.TempDir()
	specification := filepath.Join(root, "reana.yaml")
	writeFile(
		t,
		specification,
		"workflow: {type: serial}\ninputs: {options: {initfiles: [missing.yaml]}}\n",
	)
	members, err := Gather(specification)
	if err != nil || len(members) != 1 {
		t.Fatalf("unexpected non-Yadage scope: %#v, %v", members, err)
	}
}
