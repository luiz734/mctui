package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type modelAwait struct {
	sucessModel tea.Model
	failModel   tea.Model
	awaitText   string
	timeoutText string
	width       int
	height      int
	spinner     spinner.Model
	timer       timer.Model
	help        help.Model
}

func InitialAwaitModel(height, width int, awaitText, timeoutText string) modelAwait {

	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	t := timer.NewWithInterval(3*time.Second, time.Millisecond)

	return modelAwait{
		awaitText:   awaitText,
		timeoutText: timeoutText,
		width:       width,
		height:      height,
		spinner:     s,
		timer:       t,
		help:        help.New(),
	}
}

func (m modelAwait) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		// Always can quit
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		// Only read input after timeout
		if !m.timer.Timedout() {
			return m, nil
		}
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen
	}

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m modelAwait) helpView() string {
	return m.help.Styles.ShortKey.Render("Press any key to quit")
}

func (m modelAwait) View() string {
	var output strings.Builder

	// Spinner and labels
	spinnerView := m.spinner.View()
	textView := m.awaitText
	if m.timer.Timedout() {
		spinnerView = ""
		textView = m.timeoutText
	}

	centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height)
	strText := fmt.Sprintf("%s %s\n", spinnerView, textView)

	// Help
	var strHelp string
	if m.timer.Timedout() {
		strHelp = m.help.Styles.FullDesc.Render(m.helpView())
	}
	both := lipgloss.JoinVertical(lipgloss.Center, centerWrapper.Render(strText), strHelp)
	output.WriteString(both)

	return output.String()
}

func (m modelAwait) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.timer.Init(),
	)
}
