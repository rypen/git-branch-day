package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type DisplayCommit struct {
	Hash    string
	Subject string
	Effort  int
	Percent float64
}

type Result struct {
	Start    string
	End      string
	Confirm  bool
	Canceled bool
}

type stage int

const (
	stageTable stage = iota
	stageForm
)

type Model struct {
	commits     []DisplayCommit
	totalEffort int
	table       table.Model
	form        *huh.Form
	startValue  string
	endValue    string
	confirm     bool
	stage       stage
	errMsg      string
	done        bool
	canceled    bool
	width       int
	height      int
}

func New(commits []DisplayCommit, totalEffort int) *Model {
	columns := []table.Column{
		{Title: "Hash", Width: 10},
		{Title: "Subject", Width: 50},
		{Title: "Effort", Width: 8},
		{Title: "Percent", Width: 9},
	}
	rows := make([]table.Row, 0, len(commits)+1)
	for _, commit := range commits {
		rows = append(rows, table.Row{
			commit.Hash,
			commit.Subject,
			fmt.Sprintf("%d", commit.Effort),
			fmt.Sprintf("%.1f%%", commit.Percent*100),
		})
	}
	rows = append(rows, table.Row{"", "TOTAL", fmt.Sprintf("%d", totalEffort), "100.0%"})
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
	)
	model := &Model{
		commits:     commits,
		totalEffort: totalEffort,
		table:       t,
		stage:       stageTable,
	}
	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Start time (HH:MM)").Value(&model.startValue),
			huh.NewInput().Title("End time (HH:MM)").Value(&model.endValue),
			huh.NewConfirm().Title("Rewrite git history with these times?").Value(&model.confirm),
		),
	)
	return model
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			if m.stage == stageTable {
				m.stage = stageForm
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
	}

	if m.stage == stageForm {
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.form.Update(msg)
		m.form = model.(*huh.Form)
		if m.form.State == huh.StateCompleted {
			m.done = true
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	if m.stage == stageTable {
		lines := []string{
			m.table.View(),
			"",
			"Press Enter to continue, Ctrl+C to cancel",
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{m.form.View()}
	if m.errMsg != "" {
		lines = append(lines, "", m.errMsg)
	}
	return strings.Join(lines, "\n")
}

func Run(commits []DisplayCommit, totalEffort int) (Result, error) {
	model := New(commits, totalEffort)
	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return Result{}, err
	}
	final := finalModel.(*Model)
	return Result{
		Start:    final.startValue,
		End:      final.endValue,
		Confirm:  final.confirm,
		Canceled: final.canceled,
	}, nil
}
