package app

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"mctui/cli"
	"mctui/colors"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Login screen
// User must input credentials before use the application
// Uses JWT to keep the user logged in
type loginModel struct {
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusUsername bool
	width         int
	height        int
	err           error
}

// Send after login attempt
type authMsg struct {
	jwtToken string
	sucess   bool
	err      error
}

func InitialLoginModel() loginModel {
	ui := textinput.New()
	ui.Placeholder = "username"
	ui.Focus()
	ui.CharLimit = 8
	ui.Width = 8
	ui.Prompt = "  "
	ui.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Surface1))
	ui.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Pink))

	pi := textinput.New()
	pi.Placeholder = "********"
	pi.CharLimit = 8
	pi.Width = 8
	pi.Prompt = "  "
	pi.EchoMode = textinput.EchoPassword
	pi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Surface1))
	pi.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Pink))

	return loginModel{
		usernameInput: ui,
		passwordInput: pi,
		focusUsername: true,
		err:           nil,
	}
}

func (m loginModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.ClearScreen)
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			username := m.usernameInput.Value()
			password := m.passwordInput.Value()

			// Focus password if still empty
			if m.focusUsername && password == "" {
				m.passwordInput.Focus()
				m.usernameInput.Blur()
				m.focusUsername = !m.focusUsername
				return m, nil
			}

			return m, func() tea.Msg {
				// Returns an authMsg
				return authenticateUser(username, password)
			}
		case tea.KeyTab:
			if m.focusUsername {
				m.passwordInput.Focus()
				m.usernameInput.Blur()
			} else {
				m.usernameInput.Focus()
				m.passwordInput.Blur()
			}
			m.focusUsername = !m.focusUsername
		}

	case authMsg:
		if msg.err != nil {
			log.Printf("Can't login: %v", msg.err)
			return m, tea.Quit
		}
		// Clear forms now. Then, when session expires, it is already clear
		m.usernameInput.Focus()
		m.usernameInput.SetValue("")
		m.passwordInput.Blur()
		m.passwordInput.SetValue("")
		m.focusUsername = true

		// Authentication works
		if msg.sucess {
			newModel := InitialCommandModel(m, msg.jwtToken, m.width, m.height)
			// Init is called when on tea.NewProgram()
			// Since we are initializing it by ourself, we need to trigger it manually
			return newModel, newModel.Init()
		}
		// Bad credentials
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen
	}

	if m.focusUsername {
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m loginModel) View() string {
	labelStye := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Pink))
	usernameLabel := labelStye.Render(fmt.Sprintf("%s", "username"))
	username := fmt.Sprintf("%s%s", usernameLabel, m.usernameInput.View())

	passwordLabel := labelStye.Render(fmt.Sprintf("%s", "password"))
	password := fmt.Sprintf("%s%s", passwordLabel, m.passwordInput.View())

	var style = lipgloss.NewStyle().
		// Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colors.Surface0)).
		Foreground(lipgloss.Color(colors.Text)).
		Padding(1).
		PaddingLeft(2).
		PaddingRight(2).
		Align(lipgloss.Center)
	both := lipgloss.JoinVertical(lipgloss.Center, username, password)
	centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height - 3)

	return fmt.Sprintf("%s\n", centerWrapper.Render(style.Render(both)))
}

func authenticateUser(username, password string) tea.Msg {
	data := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second, // Timeout for connection setup
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	url := fmt.Sprintf(cli.Args.Address("login"))

	log.Printf("Making request to %s", url)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		var errMsg error
		if os.IsTimeout(err) {
			errMsg = fmt.Errorf("timeout error: %w", err)
		} else {
			errMsg = fmt.Errorf("error making request: %w", err)
		}
		return authMsg{
			err: errMsg,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	trimmed := strings.TrimSpace(string(body))
	return authMsg{jwtToken: trimmed, sucess: resp.Status == "200 OK"}
}
