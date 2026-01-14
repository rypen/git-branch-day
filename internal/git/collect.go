package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	Hash       string
	Subject    string
	AuthorDate time.Time
	Added      int
	Deleted    int
	Effort     int
}

func CollectTodayCommits(repoPath string, now time.Time) ([]Commit, error) {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	logOut, err := gitOutput(repoPath, "log",
		"--since="+startOfDay.Format(time.RFC3339),
		"--until="+endOfDay.Format(time.RFC3339),
		"--reverse",
		"--date=iso-strict",
		`--pretty=format:%H%x1f%ad%x1f%s%x1e`,
	)
	if err != nil {
		return nil, err
	}

	entries := strings.Split(strings.TrimSpace(logOut), "\x1e")
	commits := make([]Commit, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "\x1f")
		if len(parts) < 3 {
			return nil, errors.New("unexpected git log output")
		}
		authorDate, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		commit := Commit{
			Hash:       strings.TrimSpace(parts[0]),
			AuthorDate: authorDate,
			Subject:    strings.TrimSpace(parts[2]),
		}
		added, deleted, effort, err := commitEffort(repoPath, commit.Hash)
		if err != nil {
			return nil, err
		}
		commit.Added = added
		commit.Deleted = deleted
		commit.Effort = effort
		commits = append(commits, commit)
	}

	return commits, nil
}

func commitEffort(repoPath, hash string) (int, int, int, error) {
	out, err := gitOutput(repoPath, "show", "--numstat", "--format=", hash)
	if err != nil {
		return 0, 0, 0, err
	}
	var addedTotal int
	var deletedTotal int
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		addedTotal += parseNumstat(parts[0])
		deletedTotal += parseNumstat(parts[1])
	}
	return addedTotal, deletedTotal, addedTotal + deletedTotal, nil
}

func parseNumstat(value string) int {
	value = strings.TrimSpace(value)
	if value == "-" {
		return 0
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return num
}

func gitOutput(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", errors.New(strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}
