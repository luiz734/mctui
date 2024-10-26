package app

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mctui/cli"
	"mctui/colors"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var useHighPerformanceRenderer = false

// Main screen
// Displays a history and a command prompt
type commandModel struct {
	history []commandOutputMsg
	// Used to fill the input wuen user press Up/Down arrows
	historyIndex int
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

type sessionExpiredMsg string

func InitialCommandModel(prevModel tea.Model, jwtToken string, width, height int) commandModel {
	ci := textinput.New()
	ci.Placeholder = "e.g. kill player1"
	ci.Focus()
	ci.CharLimit = 128
	ci.Prompt = "> "
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Surface1)
	ci.PromptStyle = lipgloss.NewStyle().Foreground(colors.Pink)

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
	log.Printf("Command Initilized with size %d %d", m.width, m.height)
	return tea.Batch(
		textinput.Blink,
		tea.ClearScreen,
		func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
}

// Break strings with len > chunkSize into multiple strings
// e.g. "foobar", chunkSize=2 becomes ["fo", "ob", "ar"]
func chunkString(s string, chunkSize int) []string {
	// Multiline responses may behave weird
	// We could do some fancy logic to handle it
	// but remove newlines is good enough
	s = strings.ReplaceAll(s, "\n", " ")
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

func (e *commandOutputMsg) View(windowWidth int) string {
	commandStyle := lipgloss.NewStyle().
		Foreground(colors.Pink).
		Bold(true)
	commandStr := commandStyle.Render(e.command)

	outputStyle := lipgloss.NewStyle().
		Foreground(colors.Surface2)
		// Background(colors.Surface0)
	chunked := strings.Join(chunkString(e.output, windowWidth), "\n")
	outputStr := outputStyle.Render(chunked)

	both := fmt.Sprintf("%s\n%s\n", commandStr, outputStr)
	return both

}

func isTask(command string) bool {
	return strings.HasPrefix(command, "!")
}

func clamp(i, minVal, maxVal int) int {
	if i < minVal {
		return minVal
	} else if i > maxVal {
		return maxVal
	}
	return i
}

func (m commandModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

			// Clear the screen
		case tea.KeyCtrlL:
			m.history = nil
			m = m.updateViewportContent()
			m.commandInput.SetValue("")

			// Fill the input with previous commands
		case tea.KeyUp:
			m.historyIndex = clamp(m.historyIndex+1, 0, len(m.history))
			if m.historyIndex > 0 {
				m.commandInput.SetValue(m.history[len(m.history)-m.historyIndex].command)
				if m.commandInput.Value() == "<empty>" {
					m.commandInput.SetValue("")
				}
				m.commandInput.SetCursor(m.commandInput.CharLimit)
			}
		case tea.KeyDown:
			m.historyIndex = clamp(m.historyIndex-1, 0, len(m.history))
			if m.historyIndex > 0 {
				m.commandInput.SetValue(m.history[len(m.history)-m.historyIndex].command)
				if m.commandInput.Value() == "<empty>" {
					m.commandInput.SetValue("")
				}
			} else if m.historyIndex == 0 {
				m.commandInput.SetValue("")
				m.commandInput.SetCursor(m.commandInput.CharLimit)
			}

		case tea.KeyEnter:
			log.Printf("User input: %s", m.commandInput.Value())
			userCmd := m.commandInput.Value()
			m.historyIndex = 0

			// Quick hack. Windows doesn't like f1 shortcut
			if userCmd == "!restore" {
				m.commandInput.SetValue("")
				newModel := InitialBackupModel(m, m.jwtToken, m.width, m.height)
				return newModel, newModel.Init()
			}

			m.commandInput.SetValue("")
			taskCmd := parseCommand(m, userCmd, m.jwtToken)

			// Tasks may take some time
			// Change to the awaitModel
			if isTask(userCmd) {
				msgLoading := fmt.Sprintf("Waiting for task %s", userCmd)
				msgDone := fmt.Sprintf("Task %s done!", userCmd)
				awaitModel := InitialAwaitModel(m, taskCmd, m.width, m.height, msgLoading, msgDone)
				cmd := awaitModel.Init()
				return awaitModel, cmd
			}
			return m, taskCmd

		case tea.KeyF1:
			newModel := InitialBackupModel(m, m.jwtToken, m.width, m.height)
			return newModel, newModel.Init()
		}

	case commandOutputMsg:
		m.history = append(m.history, msg)
		m = m.updateViewportContent()

	// We get the message forwarded from awaitModel
	case taskFinishedMsg:
		m.history = append(m.history, commandOutputMsg{
			command: msg.title,
			output:  msg.msg,
		})
		log.Printf("Append task %s to history", msg.title)
		m = m.updateViewportContent()

	// Go back to login screen
	case sessionExpiredMsg:
		return m.prevModel.Update(nil)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.commandInput.Width = m.width
		marginVertical := lipgloss.Height(m.promptView())

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-marginVertical)
			// Remove default keymaps
			m.viewport.KeyMap = viewport.KeyMap{}
			m.viewport.MouseWheelEnabled = true
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - marginVertical
		}
		m = m.updateViewportContent()
	}

	m.commandInput, cmd = m.commandInput.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m commandModel) updateViewportContent() commandModel {
	m.viewport.SetContent(m.HistoryView())
	if m.viewport.TotalLineCount() > m.height {
		m.viewport.GotoBottom()
	}
	return m
}

func (m commandModel) HistoryView() string {
	var lines strings.Builder
	for _, command := range m.history {
		line := command.View(m.width)
		lines.WriteString(line)
		lines.WriteString("\n")
	}
	lines.WriteString("\n")

	return lines.String()
}

func (m commandModel) promptView() string {
	labelStye := lipgloss.NewStyle().Foreground(colors.Pink)
	commandLabel := labelStye.Render(fmt.Sprintf("%s", "command"))
	commandView := fmt.Sprintf("%s%s", commandLabel, m.commandInput.View())
	commandView = lipgloss.NewStyle().Margin(1, 0, 0, 0).Render(commandView)
	return commandView
}

func (m commandModel) View() string {
	both := lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		m.promptView())
	return fmt.Sprintf("%s", both)
}

func parseCommand(m tea.Model, command string, jwtToken string) tea.Cmd {
	if strings.HasPrefix(command, "!") {
		withoutPrefix := command[1:]
		switch command {
		case "!backup":
			return requestMakeBackup(jwtToken)
		default:
			return requestSendTask(withoutPrefix, jwtToken)
		}
	}

	log.Printf("Not a task. Skip await screen later")
	return requestSendCommand(command, jwtToken)
}

// Tasks starts with !
// e.g. !start !stop
func requestSendTask(taskName, jwtToken string) tea.Cmd {
	return func() tea.Msg {
		data := map[string]string{"task": taskName}
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Error marshalling JSON: %v", err)
		}

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: transport}

		url := fmt.Sprintf(cli.Args.Address("task"))

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

		// if resp.StatusCode != 200 {
		// 	log.Printf("Session expired: login again")
		// 	return sessionExpiredMsg("Session expired: login again")
		// }
		body, err := io.ReadAll(resp.Body)
		// Response may contain newlines or spaces
		// Who knows
		body = []byte(strings.Trim(string(body), " \n"))

		var msg taskFinishedMsg
		msg.title = "!" + taskName
		msg.msg = fmt.Sprintf("%d %s", resp.StatusCode, body)
		msg.sucess = true
		if resp.StatusCode != 200 {
			msg.msg = fmt.Sprintf("%s", body)
			msg.sucess = false
		}

		return msg
	}
}

func requestSendCommand(command, jwtToken string) tea.Cmd {
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
			log.Printf("Bad command: %s", command)
			// return sessionExpiredMsg("session expired: login again")
		}
		body, err := io.ReadAll(resp.Body)

		if command == "help" && resp.StatusCode == 200 {
			return commandOutputMsg{command, cleanHelpOutput(string(body))}
		}
		if command == "" {
			command = "<empty>"
		}
		return commandOutputMsg{command, string(body)}
	}
}

func cleanHelpOutput(output string) string {
	var parsedBuilder strings.Builder

	lines := strings.Split(output, "/")
	for _, line := range lines {
		withoutArgs := strings.SplitN(line, " ", 2)
		parsedBuilder.WriteString(fmt.Sprintf("%s, ", withoutArgs[0]))
	}

	trimCommas := strings.Trim(parsedBuilder.String(), " , ")
	// return parsedBuilder.String()
	return trimCommas
}
