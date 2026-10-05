/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// Logout holds this lock for as long as its own revocation HTTP call
	// takes (NewHTTPClient's Timeout, also 30s) -- keep a margin here so a
	// concurrent waiter's patience doesn't expire at the same instant a
	// slow-but-legitimate revocation finishes.
	credentialLockTimeout = 35 * time.Second
	lockPollInterval      = 100 * time.Millisecond
)

// openLockFile opens (creating if needed) a lock file without following a
// symlink. A predictable lock path in a shared directory would otherwise let
// a local attacker pre-create it as a symlink or hard link and make the
// client lock, and chmod, a file of the attacker's choosing.
func openLockFile(path string) (*os.File, error) {
	unsafe := func(reason string) error {
		return fmt.Errorf(
			"REANA client credential lock path %s %s; remove it and retry, "+
				"and keep the client configuration in a private directory",
			path,
			reason,
		)
	}
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW,
		0o600,
	)
	if errors.Is(err, unix.ELOOP) {
		return nil, unsafe("is a symbolic link")
	}
	if errors.Is(err, unix.EISDIR) {
		return nil, unsafe("is not a regular file")
	}
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	current, err := os.Lstat(path)
	if err != nil {
		file.Close()
		return nil, err
	}
	reason := "is not a regular file"
	if opened.Mode().IsRegular() && os.SameFile(opened, current) {
		reason = "cannot be inspected"
		if stat, ok := opened.Sys().(*syscall.Stat_t); ok {
			reason = lockFileOwnershipProblem(stat, os.Geteuid())
		}
	}
	if reason != "" {
		file.Close()
		return nil, unsafe(reason)
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

// lockFileOwnershipProblem rejects a lock file that another path or another
// user can also reach, before the client changes its permissions.
func lockFileOwnershipProblem(stat *syscall.Stat_t, euid int) string {
	if stat.Nlink != 1 {
		return "has multiple hard links"
	}
	if int(stat.Uid) != euid {
		return "is owned by another user"
	}
	return ""
}

func acquireLock(path string, wait bool) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := openLockFile(path)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(credentialLockTimeout)
	for {
		err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return file, nil
		}
		if !wait || time.Now().After(deadline) {
			file.Close()
			if wait {
				return nil, errors.New(
					"timed out waiting for the REANA client credential lock",
				)
			}
			return nil, nil
		}
		time.Sleep(lockPollInterval)
	}
}

func releaseLock(file *os.File) {
	if file == nil {
		return
	}
	_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
	_ = file.Close()
}
