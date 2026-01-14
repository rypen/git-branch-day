package main

import (
	"fmt"
	"os"
	"time"

	"branch-day/internal/git"
	"branch-day/internal/rewrite"
	"branch-day/internal/timeutil"
	"branch-day/internal/tui"
)

func main() {
	repoPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	now := time.Now()
	commits, err := git.CollectTodayCommits(repoPath, now)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(commits) == 0 {
		fmt.Println("No commits found for today.")
		return
	}

	displayCommits, totalEffort := buildDisplay(commits)
	result, err := tui.Run(displayCommits, totalEffort)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.Canceled || !result.Confirm {
		fmt.Println("Cancelled.")
		return
	}

	start, end, err := timeutil.TimeRangeForDay(now, result.Start, result.End)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	efforts := make([]int, len(commits))
	commitHashes := make([]string, len(commits))
	for i, commit := range commits {
		efforts[i] = commit.Effort
		commitHashes[i] = commit.Hash
	}
	times, err := timeutil.AllocateTimes(start, end, efforts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := rewrite.RebaseWithTimestamps(repoPath, commitHashes, times); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("Rebase completed.")
}

func buildDisplay(commits []git.Commit) ([]tui.DisplayCommit, int) {
	total := 0
	for _, commit := range commits {
		total += commit.Effort
	}
	display := make([]tui.DisplayCommit, len(commits))
	for i, commit := range commits {
		percent := 0.0
		if total > 0 {
			percent = float64(commit.Effort) / float64(total)
		} else if len(commits) > 0 {
			percent = 1.0 / float64(len(commits))
		}
		display[i] = tui.DisplayCommit{
			Hash:    shortHash(commit.Hash),
			Subject: commit.Subject,
			Effort:  commit.Effort,
			Percent: percent,
		}
	}
	return display, total
}

func shortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}
