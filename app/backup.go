package app

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mctui/cli"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type backup struct {
	readable, raw string
}

func (i backup) Title() string       { return i.readable }
func (i backup) Description() string { return i.raw }
func (i backup) FilterValue() string { return i.readable }

type backupModel struct {
	list     list.Model
	jwtToken string
	width    int
	height   int
}

type fetchMsg struct {
	items []list.Item
}

func fetchData(jwtToken string) tea.Cmd {
	return func() tea.Msg {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: transport}

		url := fmt.Sprintf(cli.Args.Address("backups"))
		req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte("")))
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
		log.Printf(string(body))

		var backupNames []string
		if err := json.Unmarshal(body, &backupNames); err != nil {
			panic(err)
		}

		var backups []backup
		var items []list.Item
		for _, name := range backupNames {
			backups = append(backups, backup{readable: name})
            readableDate, _ := HumanizeBackupDate(name)
            items = append(items, backup{readable: readableDate, raw: name})
			log.Printf(name)
		}

		return fetchMsg{
			items: items,
		}
	}
}
func HumanizeBackupDate(filename string) (string, error) {
	const layout = "backup-2006-01-02-15-04-05.zip"
	t, err := time.Parse(layout, filename)
	if err != nil {
		return "", err
	}
	return humanize.Time(t), nil
}

func (m backupModel) Init() tea.Cmd {
	return tea.Batch(
		fetchData(m.jwtToken),
		func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
}

func (m backupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case fetchMsg:
		m.list.SetItems(msg.items)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m backupModel) View() string {
	return docStyle.Render(m.list.View())
}

func InitialBackupModel(jwtToken string, width, height int) backupModel {
	items := []list.Item{}

	m := backupModel{
		list:     list.New(items, list.NewDefaultDelegate(), 0, 0),
		jwtToken: jwtToken,
		width:    width,
		height:   height,
	}
	m.list.Title = "Backups"

	return m
}
