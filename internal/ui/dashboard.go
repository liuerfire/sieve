package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liuerfire/sieve/internal/engine"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	sourceStyle = lipgloss.NewStyle().
			Width(25).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Width(15)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Height(8)

	highInterestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	interestStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))

	doneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
)

type sourceStatus struct {
	name    string
	current int
	total   int
	status  string // "Pending", "Fetching", "Processing", "Done", "Error"
	lastItem string
	lastLevel string
}

type Model struct {
	sources    map[string]*sourceStatus
	sourceOrder []string
	logs       []string
	spinner    spinner.Model
	quitting   bool
	done       bool
	startTime  time.Time
	totalProcessed int
	highCount      int
}

func NewModel(sourceNames []string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := Model{
		sources:     make(map[string]*sourceStatus),
		sourceOrder: sourceNames,
		logs:        make([]string, 0),
		spinner:     s,
		startTime:   time.Now(),
	}

	for _, name := range sourceNames {
		m.sources[name] = &sourceStatus{name: name, status: "Pending"}
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case engine.ProgressEvent:
		m.handleProgress(msg)
		if msg.Type == "gen_done" {
			m.done = true
			// We don't quit automatically so the user can see the final state
		}
	}

	return m, nil
}

func (m *Model) handleProgress(ev engine.ProgressEvent) {
	switch ev.Type {
	case "source_start":
		if s, ok := m.sources[ev.Source]; ok {
			s.status = "Fetching"
		}
		m.addLog(fmt.Sprintf("Fetching source: %s", ev.Source))

	case "item_start":
		if s, ok := m.sources[ev.Source]; ok {
			s.status = "Processing"
			s.current = ev.Count
			s.total = ev.Total
			s.lastItem = ev.Item
		}

	case "item_done":
		if s, ok := m.sources[ev.Source]; ok {
			s.lastLevel = ev.Level
			m.totalProcessed++
			if ev.Level == "high_interest" {
				m.highCount++
			}
		}

	case "source_done":
		if s, ok := m.sources[ev.Source]; ok {
			if ev.Message != "" {
				s.status = "Error"
				m.addLog(fmt.Sprintf("Error in %s: %s", ev.Source, ev.Message))
			} else {
				s.status = "Done"
				s.current = ev.Total
				s.total = ev.Total
			}
		}
		m.addLog(fmt.Sprintf("Finished source: %s", ev.Source))

	case "gen_start":
		m.addLog(ev.Message)

	case "gen_done":
		m.addLog(ev.Message)
	}
}

func (m *Model) addLog(msg string) {
	m.logs = append(m.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(m.logs) > 8 {
		m.logs = m.logs[len(m.logs)-8:]
	}
}

func (m Model) View() string {
	if m.quitting {
		return "Quitting...\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("Sieve Aggregator Dashboard"))
	b.WriteString("\n\n")

	// Sources
	for _, name := range m.sourceOrder {
		s := m.sources[name]

		statusIcon := m.spinner.View()
		statusTxt := statusStyle.Render(s.status)

		switch s.status {
		case "Done":
			statusIcon = doneStyle.Render("✔")
			statusTxt = doneStyle.Render(s.status)
		case "Error":
			statusIcon = errStyle.Render("✘")
			statusTxt = errStyle.Render(s.status)
		case "Pending":
			statusIcon = " "
		}

		progress := ""
		if s.total > 0 {
			progress = fmt.Sprintf("[%d/%d]", s.current, s.total)
		}

		itemInfo := ""
		if s.lastItem != "" && s.status == "Processing" {
			itemInfo = itemStyle.Render(" - " + truncate(s.lastItem, 40))
		}

		b.WriteString(fmt.Sprintf("%s %s %s %s%s\n",
			statusIcon,
			sourceStyle.Render(name),
			statusTxt,
			progress,
			itemInfo))
	}

	b.WriteString("\n")

	// Stats
	elapsed := time.Since(m.startTime).Round(time.Second)
	stats := fmt.Sprintf("Processed: %d | High Interest: %s | Time: %s",
		m.totalProcessed,
		highInterestStyle.Render(fmt.Sprintf("%d", m.highCount)),
		elapsed)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(stats))
	b.WriteString("\n\n")

	// Logs
	b.WriteString(lipgloss.NewStyle().Underline(true).Render("Recent Logs:"))
	b.WriteString("\n")
	b.WriteString(logStyle.Render(strings.Join(m.logs, "\n")))

	if m.done {
		b.WriteString("\n\n")
		b.WriteString(doneStyle.Render(" Aggregation Complete! Press 'q' to exit. "))
	} else {
		b.WriteString("\n\n")
		b.WriteString(itemStyle.Render(" Press 'q' to quit "))
	}

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func RunDashboard(ctx context.Context, sourceNames []string, runFunc func(func(engine.ProgressEvent)) error) error {
	m := NewModel(sourceNames)
	p := tea.NewProgram(m)

	errChan := make(chan error, 1)
	go func() {
		err := runFunc(func(ev engine.ProgressEvent) {
			p.Send(ev)
		})
		errChan <- err
	}()

	if _, err := p.Run(); err != nil {
		return err
	}

	return <-errChan
}
