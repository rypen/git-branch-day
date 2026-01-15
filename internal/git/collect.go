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

	return parseLogOutput(repoPath, logOut)
}

func CollectCommitsFrom(repoPath, firstSha string) ([]Commit, error) {
	resolved, err := resolveHash(repoPath, firstSha)
	if err != nil {
		return nil, err
	}
	isAncestor, err := isAncestor(repoPath, resolved, "HEAD")
	if err != nil {
		return nil, err
	}
	if !isAncestor {
		return nil, errors.New("specified commit is not an ancestor of HEAD")
	}

	logOut, err := gitOutput(repoPath, "log",
		"--reverse",
		"--date=iso-strict",
		`--pretty=format:%H%x1f%ad%x1f%s%x1e`,
		"--ancestry-path",
		resolved+"..HEAD",
	)
	if err != nil {
		return nil, err
	}
	commits, err := parseLogOutput(repoPath, logOut)
	if err != nil {
		return nil, err
	}

	firstOut, err := gitOutput(repoPath, "show",
		"-s",
		"--date=iso-strict",
		`--pretty=format:%H%x1f%ad%x1f%s%x1e`,
		resolved,
	)
	if err != nil {
		return nil, err
	}
	firstCommits, err := parseLogOutput(repoPath, firstOut)
	if err != nil {
		return nil, err
	}
	if len(firstCommits) == 0 {
		return nil, errors.New("unable to read specified commit")
	}
	return append(firstCommits, commits...), nil
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

func parseLogOutput(repoPath, logOut string) ([]Commit, error) {
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

func resolveHash(repoPath, value string) (string, error) {
	out, err := gitOutput(repoPath, "rev-parse", value)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func isAncestor(repoPath, ancestor, descendant string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
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
