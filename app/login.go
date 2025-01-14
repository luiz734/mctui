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
	"github.com/muesli/reflow/wordwrap"
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
	ui.CharLimit = 16
	ui.Width = 8
	ui.Prompt = "  "
	ui.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Surface1)
	ui.PromptStyle = lipgloss.NewStyle().Foreground(colors.Pink)

	pi := textinput.New()
	pi.Placeholder = "********"
	pi.CharLimit = 16
	pi.Width = 8
	pi.Prompt = "  "
	pi.EchoMode = textinput.EchoPassword
	pi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colors.Surface1)
	pi.PromptStyle = lipgloss.NewStyle().Foreground(colors.Pink)

	// Use for debug only
	ui.SetValue("admin")
	pi.SetValue("adminpass123")

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
		// User always can quit
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		if m.err != nil {
			// When user press any key on the error screen
			m.err = nil
			m = m.clearForm()
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEnter:
			username := m.usernameInput.Value()
			password := m.passwordInput.Value()

			// Don't change focus if username is empty
			if username == "" {
				return m, nil
			}
			// Focus password if still empty
			if m.focusUsername && password == "" {
				m.passwordInput.Focus()
				m.usernameInput.Blur()
				m.focusUsername = !m.focusUsername
				return m, nil
			}
			// Don't make the request with empty password
			if password == "" {
				return m, nil
			}
			return m, func() tea.Msg {
				// Returns an authMsg
				return requestAuthenticateUser(username, password)
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
			m.err = fmt.Errorf("Can't login: %v", msg.err)
			log.Printf("%v", m.err)
			// return m, tea.Quit
		}
		// Clear forms now. Then, when session expires, it is already clear
		m.clearForm()

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

func (m loginModel) clearForm() loginModel {
	m.usernameInput.Focus()
	m.usernameInput.SetValue("")
	m.passwordInput.Blur()
	m.passwordInput.SetValue("")
	m.focusUsername = true
	return m
}

func (m loginModel) ErrorView() string {
	centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height - 2)

	errTitle := lipgloss.NewStyle().
		Foreground(colors.Pink).
		Bold(true).
		Render("\n\nError trying to login\n")

	errDescription := lipgloss.NewStyle().
		Foreground(colors.Surface2).
		Align(lipgloss.Center).
		Render(wordwrap.String(fmt.Sprintf("\n%v", m.err), m.width-12))
		// Render("bar")

	both := lipgloss.JoinVertical(lipgloss.Center, errTitle, errDescription)
	return centerWrapper.Render(both)
}

func (m loginModel) View() string {
	if m.err != nil {
		return m.ErrorView()
	}

	centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height - 3)

	labelStye := lipgloss.NewStyle().Foreground(colors.Pink)
	usernameLabel := labelStye.Render(fmt.Sprintf("%s", "username"))
	username := fmt.Sprintf("%s%s", usernameLabel, m.usernameInput.View())

	passwordLabel := labelStye.Render(fmt.Sprintf("%s", "password"))
	password := fmt.Sprintf("%s%s", passwordLabel, m.passwordInput.View())

	var style = lipgloss.NewStyle().
		// Border(lipgloss.NormalBorder()).
		BorderForeground(colors.Surface0).
		Foreground(colors.Text).
		Padding(1).
		PaddingLeft(2).
		PaddingRight(2).
		Align(lipgloss.Center)
	both := lipgloss.JoinVertical(lipgloss.Center, username, password)
	return fmt.Sprintf("%s\n", centerWrapper.Render(style.Render(both)))
}

func requestAuthenticateUser(username, password string) tea.Msg {
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
