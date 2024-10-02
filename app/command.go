package app

import (
	"fmt"
	"strings"

	"mctui/colors"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type rconEntry struct {
	command string
	output  string
}

func (e *rconEntry) View() string {
	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Pink)).
		Bold(true)
	commandStr := commandStyle.Render(e.command)

	outputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Surface2))
	outputStr := outputStyle.Render(e.output)

	both := fmt.Sprintf("%s\n%s\n", commandStr, outputStr)
	return both

}

type commandModel struct {
	history      []rconEntry
	commandInput textinput.Model
	width        int
	height       int
	err          error
}

type outputMsg struct {
	c string
	o string
}

func InitialCommandModel() commandModel {
	ci := textinput.New()
	ci.Placeholder = "e.g. /kill player1"
	ci.Focus()
	ci.CharLimit = 128
	// ui.Width = 8
	ci.Prompt = "> "
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Surface1))
	ci.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Pink))

	return commandModel{
		commandInput: ci,
		err:          nil,
	}
}

func (m commandModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.ClearScreen)
}

func (m commandModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			cmd = sendCommand(m.commandInput.Value())
			m.commandInput.SetValue("")
			return m, cmd
		}
		switch msg.String() {
		}

	case errMsg:
		m.err = msg
		return m, nil
	case outputMsg:
		m.history = append(m.history, rconEntry{
			command: msg.c,
			output:  msg.o,
		})
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.commandInput.Width = m.width
		return m, tea.ClearScreen
	}

	m.commandInput, cmd = m.commandInput.Update(msg)
	return m, cmd
}

func (m commandModel) View() string {
	labelStye := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Pink))
	commandLabel := labelStye.Render(fmt.Sprintf("%s", "command"))
	commandView := fmt.Sprintf("%s%s", commandLabel, m.commandInput.View())

	var lines strings.Builder
	for _, command := range m.history {
		line := command.View()
		lines.WriteString(line)
        lines.WriteString("\n")
	}
	lines.WriteString("\n")

	historyStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colors.Surface1)).
		Padding(1, 4).
		Width(m.width - 2).
		Height(m.height - 4)

	// var style = lipgloss.NewStyle().
	// 	// Border(lipgloss.NormalBorder()).
	// 	BorderForeground(lipgloss.Color(colors.Surface0)).
	// 	Foreground(lipgloss.Color(colors.Text)).
	// 	Padding(1).
	// 	PaddingLeft(2).
	// 	PaddingRight(2).
	// 	Align(lipgloss.Center)
	both := lipgloss.JoinVertical(lipgloss.Left,
		historyStyle.Render(lines.String()),
		commandView)
	// centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height - 3)

	// return fmt.Sprintf("%s\n", centerWrapper.Render(style.Render(both)))
	return fmt.Sprintf("%s\n", both)
}

func sendCommand(command string) tea.Cmd {
	return func() tea.Msg {
		return outputMsg{command, "this is output"}
	}
}
