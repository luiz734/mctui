package app

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusUsername bool
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
	ui.CharLimit = 20
	ui.Width = 20
	ui.Prompt = ": "

	pi := textinput.New()
	pi.Placeholder = "********"
	pi.CharLimit = 20
	pi.Width = 20
	pi.Prompt = ": "
	pi.EchoMode = textinput.EchoPassword

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
	}

	if m.focusUsername {
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"username%s\npassword%s\n",
		m.usernameInput.View(),
		m.passwordInput.View(),
	) + "\n"
}
