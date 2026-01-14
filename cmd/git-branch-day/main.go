package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"branch-day/internal/git"
	"branch-day/internal/rewrite"
	"branch-day/internal/timeutil"
	"branch-day/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		return
	}
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
	endDefault := now.Format("15:04")
	result, err := tui.Run(displayCommits, totalEffort, "", endDefault)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.Canceled {
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
	fmt.Print(formatPreview(commits, times, start))
	if !promptConfirm() {
		fmt.Println("Cancelled.")
		return
	}

	if err := rewrite.RebaseWithTimestamps(repoPath, commitHashes, times); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("Rebase completed.")
}

const version = "0.1.0"

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

func formatPreview(commits []git.Commit, times []time.Time, start time.Time) string {
	var builder strings.Builder
	builder.WriteString("Planned commit timeline:\n")
	totalEffort := 0
	for _, commit := range commits {
		totalEffort += commit.Effort
	}
	builder.WriteString("End Time          Duration  Effort  Percent  Hash     Subject\n")
	prev := start
	for i, commit := range commits {
		end := times[i]
		duration := end.Sub(prev)
		prev = end
		percent := 0.0
		if totalEffort > 0 {
			percent = float64(commit.Effort) / float64(totalEffort)
		} else if len(commits) > 0 {
			percent = 1.0 / float64(len(commits))
		}
		builder.WriteString(fmt.Sprintf(
			"%-16s  %-8s  %-6d  %-7s  %-7s  %s\n",
			end.Format("2006-01-02 15:04"),
			formatDuration(duration),
			commit.Effort,
			fmt.Sprintf("%.1f%%", percent*100),
			shortHash(commit.Hash),
			commit.Subject,
		))
	}
	return builder.String()
}

func formatDuration(value time.Duration) string {
	totalMinutes := int(value.Round(time.Minute).Minutes())
	if totalMinutes < 0 {
		totalMinutes = 0
	}
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh%02dm", hours, minutes)
}

func promptConfirm() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Rewrite git history with these times? [y/N]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		value := strings.TrimSpace(strings.ToLower(input))
		if value == "" || value == "n" || value == "no" {
			return false
		}
		if value == "y" || value == "yes" {
			return true
		}
	}
}
