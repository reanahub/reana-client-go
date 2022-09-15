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
	"runtime"
	"runtime/pprof"

	"github.com/spf13/pflag"
)

var (
	profileMode         string
	profileOutputFormat string = "profile-%s.pprof"
)

func addProfilerFlags(flags *pflag.FlagSet) {
	flags.StringVar(
		&profileMode,
		"profile",
		"none",
		"Enable profiling. One of (none|cpu|heap)",
	)
}

func setupProfiler() error {
	var (
		f   *os.File
		err error
	)
	switch profileMode {
	case "none":
		return nil
	case "cpu":
		profileOutput := fmt.Sprintf(profileOutputFormat, profileMode)
		f, err = os.Create(profileOutput)
		if err != nil {
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return err
		}
	default:
		// Check if the profile mode is valid.
		if profile := pprof.Lookup(profileMode); profile == nil {
			return fmt.Errorf("unknown profile '%s'", profileMode)
		}
	}
	return nil
}

func stopProfiler() error {
	switch profileMode {
	case "none":
		return nil
	case "cpu":
		pprof.StopCPUProfile()
	case "heap":
		runtime.GC()
		fallthrough
	default:
		profile := pprof.Lookup(profileMode)
		if profile == nil {
			return nil
		}
		profileOutput := fmt.Sprintf(profileOutputFormat, profileMode)
		f, err := os.Create(profileOutput)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := profile.WriteTo(f, 0); err != nil {
			return err
		}
	}
	return nil
}
