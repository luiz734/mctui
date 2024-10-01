package app

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Color string

var (
	Surface0 = "#313244"
	Surface1 = "#45475a"
	Surface2 = "#585b70"
	Pink     = "#f5c2e7"
	Text     = "#cdd6f4"
)

type model struct {
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusUsername bool
	width         int
	height        int
	err           error
}

type errMsg error

type loginMsg struct {
	username string
	password string
}

func InitialModel() model {
	ui := textinput.New()
	ui.Placeholder = "username"
	ui.Focus()
	ui.CharLimit = 8
	ui.Width = 8
	ui.Prompt = "  "
	ui.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(Surface1))
	ui.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(Pink))

	pi := textinput.New()
	pi.Placeholder = "********"
	pi.CharLimit = 8
	pi.Width = 8
	pi.Prompt = "  "
	pi.EchoMode = textinput.EchoPassword
	pi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(Surface1))
	pi.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(Pink))

	return model{
		usernameInput: ui,
		passwordInput: pi,
		focusUsername: true,
		err:           nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.ClearScreen)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			return m, func() tea.Msg {
				return loginMsg{
					username: m.usernameInput.Value(),
					password: m.passwordInput.Value(),
				}
			}
		}
		switch msg.String() {

		case "tab":
			if m.focusUsername {
				m.passwordInput.Focus()
				m.usernameInput.Blur()
			} else {
				m.usernameInput.Focus()
				m.passwordInput.Blur()
			}

			m.focusUsername = !m.focusUsername
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	case loginMsg:
		log.Printf("%s %s", msg.username, msg.password)
		return m, tea.Quit
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

func (m model) View() string {
	labelStye := lipgloss.NewStyle().Foreground(lipgloss.Color(Pink))
    usernameLabel := labelStye.Render(fmt.Sprintf("%s", "username"))
	username :=  fmt.Sprintf("%s%s", usernameLabel, m.usernameInput.View())

    passwordLabel := labelStye.Render(fmt.Sprintf("%s", "password"))
	password :=  fmt.Sprintf("%s%s", passwordLabel, m.passwordInput.View())

	var style = lipgloss.NewStyle().
		// Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(Surface0)).
		Foreground(lipgloss.Color(Text)).
		Padding(1).
		PaddingLeft(2).
		PaddingRight(2).
		Align(lipgloss.Center)
	both := lipgloss.JoinVertical(lipgloss.Center, username, password)
	centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height - 3)

	return fmt.Sprintf("%s\n", centerWrapper.Render(style.Render(both)))
}
