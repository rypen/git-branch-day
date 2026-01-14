package rewrite

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func RebaseWithTimestamps(repoPath string, commitHashes []string, times []time.Time) error {
	if len(commitHashes) == 0 {
		return errors.New("no commits to rewrite")
	}
	if len(commitHashes) != len(times) {
		return errors.New("commit and time counts do not match")
	}

	timesFile, scriptFile, cleanup, err := prepareRebaseFiles(times)
	if err != nil {
		return err
	}
	defer cleanup()

	base, useRoot, err := baseCommit(repoPath, commitHashes[0])
	if err != nil {
		return err
	}

	args := []string{"rebase", "--rebase-merges", "--exec", scriptFile}
	if useRoot {
		args = append(args, "--root")
	} else {
		args = append(args, base)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), "GIT_BRANCH_DAY_TIMES="+timesFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return errors.New(strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}

func baseCommit(repoPath, hash string) (string, bool, error) {
	cmd := exec.Command("git", "rev-parse", hash+"^")
	cmd.Dir = repoPath
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", true, nil
	}
	return strings.TrimSpace(stdout.String()), false, nil
}

func prepareRebaseFiles(times []time.Time) (string, string, func(), error) {
	dir, err := os.MkdirTemp("", "branch-day-")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}

	timesPath := filepath.Join(dir, "times.txt")
	var lines []string
	for _, t := range times {
		lines = append(lines, t.Format(time.RFC3339))
	}
	if err := os.WriteFile(timesPath, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		cleanup()
		return "", "", nil, err
	}

	scriptPath := filepath.Join(dir, "rewrite.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -e",
		"list=\"$GIT_BRANCH_DAY_TIMES\"",
		"if [ ! -f \"$list\" ]; then",
		"  echo \"missing times file\" >&2",
		"  exit 1",
		"fi",
		"ts=$(sed -n '1p' \"$list\")",
		"if [ -z \"$ts\" ]; then",
		"  echo \"no more timestamps\" >&2",
		"  exit 1",
		"fi",
		"tmp=\"${list}.tmp\"",
		"tail -n +2 \"$list\" > \"$tmp\"",
		"mv \"$tmp\" \"$list\"",
		"GIT_AUTHOR_DATE=\"$ts\" GIT_COMMITTER_DATE=\"$ts\" git commit --amend --no-edit --date \"$ts\" >/dev/null",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil {
		cleanup()
		return "", "", nil, err
	}

	return timesPath, scriptPath, cleanup, nil
}
