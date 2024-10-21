package app

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type modelAwait struct {
	prevModel   tea.Model
	task        tea.Cmd
	taskMsg     taskFinishedMsg
	loadingText string
	timeoutText string
	width       int
	height      int
	done        bool
	spinner     spinner.Model
	timer       timer.Model
	help        help.Model
}

type taskFinishedMsg struct {
	title  string
	msg    string
	sucess bool
	async  bool
}

func InitialAwaitModel(
	prevModel tea.Model,
	task tea.Cmd,
	width, height int,
	loadingText, timeoutText string,
) modelAwait {

	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	t := timer.NewWithInterval(3*time.Second, time.Millisecond)

	return modelAwait{
		prevModel:   prevModel,
		task:        task,
		loadingText: loadingText,
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
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		// Forward the finish notification
		// Parent may want to know what happens
		if m.done {
			return m.prevModel, func() tea.Msg { return m.taskMsg }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen
	case taskFinishedMsg:
		log.Printf("Task %s done", msg.title)
		m.taskMsg = msg
		m.done = true
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
	textView := m.loadingText
	var errDetails string
	if m.done {
		textView = fmt.Sprintf("Task complete!")
		spinnerView = ":) "
		if !m.taskMsg.sucess {
			errDetails = m.taskMsg.msg
			spinnerView = ":( "
			textView = fmt.Sprintf("Task failed!")
		}
	}

	strErr := m.help.Styles.ShortKey.Render(errDetails)

	centerWrapper := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Width(m.width - 2).Height(m.height)
	strText := fmt.Sprintf("%s %s\n%s", spinnerView, textView, strErr)

	// Help
	var strHelp string
	if m.done {
		strHelp = m.help.Styles.FullDesc.Render(m.helpView())
	}
	both := lipgloss.JoinVertical(lipgloss.Center, centerWrapper.Render(strText), strHelp)
	output.WriteString(both)

	return output.String()
}

func (m modelAwait) Init() tea.Cmd {
	log.Printf("Enter awaitModel.Init()")
	return tea.Batch(
		m.spinner.Tick,
		m.timer.Init(),
		m.task,
	)
}
