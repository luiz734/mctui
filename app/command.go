package app

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"log"
	"mctui/cli"
	"mctui/colors"
	"net/http"
	"strings"
)

var useHighPerformanceRenderer = true

// Main screen
// Displays a history and a command prompt
type commandModel struct {
	history      []commandOutputMsg
	commandInput textinput.Model
	viewport     viewport.Model
	prevModel    tea.Model
	jwtToken     string
	ready        bool
	width        int
	height       int
	err          error
}

// Send after rcon commands, tasks
// Can also display error values
// Used to append to history
type commandOutputMsg struct {
	command string
	output  string
}

// type rconEntry struct {
// 	command string
// 	output  string
// }

type sessionExpiredMsg string

func (e *commandOutputMsg) View() string {
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

func InitialCommandModel(prevModel tea.Model, jwtToken string, width, height int) commandModel {
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
		jwtToken:     jwtToken,
		width:        width,
		height:       height,
		prevModel:    prevModel,
	}
}

func (m commandModel) Init() tea.Cmd {
	// m.viewport = viewport.New(m.width, m.height-6)
	// m.viewport.MouseWheelEnabled = true
	// // m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
	// m.ready = true

	log.Printf("Command Initilized with size %d %d", m.width, m.height)
	return tea.Batch(
		textinput.Blink,
		tea.ClearScreen,
		func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
}

func isTask(command string) bool {
	return strings.HasPrefix(command, "!")
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
			log.Printf("User input: %s", m.commandInput.Value())
			userCmd := m.commandInput.Value()

			// Quick hack. Windows doesn't like f1 shortcut
			if userCmd == "!restore" {
				m.commandInput.SetValue("")
				newModel := InitialBackupModel(m, m.jwtToken, m.width, m.height)
				return newModel, newModel.Init()
			}

			m.commandInput.SetValue("")
			taskCmd, err := parseCommand(m, userCmd, m.jwtToken)
			// Input start with ! but invalid
			// We handle that
			if err != nil {
				log.Printf("%v", err)
				return m, taskCmd
			}
			// Command not start with !
			// RCON will handle that
			if isTask(userCmd) {
				awaitModel := InitialAwaitModel(m, taskCmd, m.width, m.height, "Making backup", "Backup done!")
				cmd := awaitModel.Init()
				return awaitModel, cmd
			}
			return m, taskCmd

		case tea.KeyCtrlJ:
			m.viewport.YOffset += 3
		case tea.KeyCtrlK:
			m.viewport.YOffset -= 3
		case tea.KeyF1:
			newModel := InitialBackupModel(m, m.jwtToken, m.width, m.height)
			return newModel, newModel.Init()
		}

	case commandOutputMsg:
		m.history = append(m.history, msg)
		// m.history = append(m.history, commandOutputMsg{
		// 	command: msg.command,
		// 	output:  msg.output,
		// })
		// return m, nil
	// case restoreBackupMsg:
	// 	if msg.status < 0 {
	// 		return m, nil
	// 	}
	// 	m.history = append(m.history, rconEntry{
	// 		command: msg.command,
	// 		output:  msg.body,
	// 	})
	// 	log.Printf("Backup restored")
	// 	return m, nil

	// We get the message forwarded from awaitModel
	case taskFinishedMsg:
		m.history = append(m.history, commandOutputMsg{
			command: msg.title,
			output:  msg.msg,
		})
		log.Printf("Append task %s to history", msg.title)
		return m, nil

	// Go back to login screen
	case sessionExpiredMsg:
		return m.prevModel.Update(nil)

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
	// if !m.ready {
	// 	return "Initializing..."
	// }
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

	viewportStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colors.Surface1)).
		Padding(1, 4).
		Width(m.width - 2).
		Height(m.height - 4)
	m.viewport.SetContent(historyStyle.Render(lines.String()))

	both := lipgloss.JoinVertical(lipgloss.Left,
		viewportStyle.Render(m.viewport.View()),
		commandView)
	return fmt.Sprintf("%s\n", both)
}

func parseCommand(m tea.Model, command string, jwtToken string) (tea.Cmd, error) {
	if strings.HasPrefix(command, "!") {
		withoutPrefix := command[1:]
		switch withoutPrefix {
		case "backup":
			return requestMakeBackup(jwtToken), nil
		default:
			return func() tea.Msg {
				return taskFinishedMsg{
					title:  "Unknown task",
					msg:    fmt.Sprintf("Task %s not valid", command),
					sucess: false,
					async:  false,
				}
			}, fmt.Errorf("Unknown task %s", command)
		}
	}
	// Remove dangerous commands e.g. "stop" or "list stop"
	if !commandIsSafe(command) {
		return func() tea.Msg {
			return commandOutputMsg{command, "You can't stop the server"}
		}, nil
	}
	log.Printf("Not a task. Skip await screen later")
	return sendCommand(command, jwtToken), nil
}
func sendCommand(command, jwtToken string) tea.Cmd {
	return func() tea.Msg {
		data := map[string]string{"command": command}
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Error marshalling JSON: %v", err)
		}

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: transport}

		// Handle special commands that starts with !
		// e.g. !backup
		url := fmt.Sprintf(cli.Args.Address("command"))

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

		resp, err := client.Do(req)
		if err != nil {
			panic(err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("session expired: login again")
			return sessionExpiredMsg("session expired: login again")
		}
		body, err := io.ReadAll(resp.Body)

		if command == "help" {
			return commandOutputMsg{command, parseHelpOutput(string(body))}
		}
		return commandOutputMsg{command, string(body)}
	}
}

// Check for "stop" inside command
// e.g. "list stop"
func commandIsSafe(command string) bool {
	tokens := strings.Split(command, " ")
	for _, t := range tokens {
		if t == "stop" {
			return false
		}
	}
	return true
}

func parseHelpOutput(output string) string {
	var parsedBuilder strings.Builder

	lines := strings.Split(output, "/")
	for _, line := range lines {
		parsedBuilder.WriteString(fmt.Sprintf("%s\n", line))
	}

	return parsedBuilder.String()
}
