package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"mctui/colors"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var useHighPerformanceRenderer = true

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
	viewport     viewport.Model
	ready        bool
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
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			cmd = sendCommand(m.commandInput.Value())
			m.commandInput.SetValue("")
			return m, cmd
		case tea.KeyCtrlJ:
			m.viewport.YOffset += 3
		case tea.KeyCtrlK:
			m.viewport.YOffset -= 3
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
		// m.viewport.SetContent(strings.Join(m.viewport.GotoBottom(), ""))
		// m.viewport.YOffset += 1
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.commandInput.Width = m.width

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.viewport.MouseWheelEnabled = true
			// m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.ready = true
		}
		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			// cmds = append(cmds, viewport.Sync(m.viewport))
		}
		return m, tea.ClearScreen
	}

	m.commandInput, cmd = m.commandInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
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

	historyStyle := lipgloss.NewStyle()
	// Border(lipgloss.RoundedBorder()).
	// BorderForeground(lipgloss.Color(colors.Surface1)).
	// Padding(1, 4).
	// Width(m.width - 2).
	// Height(m.height - 4)

	viewportStyle := lipgloss.NewStyle().
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
	m.viewport.SetContent(historyStyle.Render(lines.String()))
	both := lipgloss.JoinVertical(lipgloss.Left,
		// historyStyle.Render(lines.String()),
		viewportStyle.Render(m.viewport.View()),
		commandView)
	// centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height - 3)

	// return fmt.Sprintf("%s\n", centerWrapper.Render(style.Render(both)))
	return fmt.Sprintf("%s\n", both)
}

func sendCommand(command string) tea.Cmd {
	return func() tea.Msg {
		data := map[string]string{"command": command}
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Error marshalling JSON: %v", err)
		}

		url := fmt.Sprintf("http://localhost:8090/command")
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			panic(err.Error())
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)

		if command == "help" {
			return outputMsg{command, parseHelpOutput(string(body))}
		}
		return outputMsg{command, string(body)}
	}
}

func parseHelpOutput(output string) string {
	var parsedBuilder strings.Builder

	lines := strings.Split(output, "/")
	for _, line := range lines {
		parsedBuilder.WriteString(fmt.Sprintf("%s\n", line))
	}

	return parsedBuilder.String()
}
