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
	list      list.Model
	jwtToken  string
	prevModel tea.Model
	width     int
	height    int
}

func InitialBackupModel(prevModel tea.Model, jwtToken string, width, height int) backupModel {
	items := []list.Item{}
	m := backupModel{
		list:      list.New(items, list.NewDefaultDelegate(), 0, 0),
		prevModel: prevModel,
		jwtToken:  jwtToken,
		width:     width,
		height:    height,
	}
	m.list.Title = "Backups"
	return m
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
		switch msg.Type {
		case tea.KeyEscape:
			log.Printf("Escape")
			errMsg := fmt.Errorf("Operation canceled by user")
			// Don't return m.prevMode.Update(msg)
			return m.prevModel, func() tea.Msg {
				return taskFinishedMsg{
					title:  "Restore backup",
					msg:    errMsg.Error(),
					sucess: false,
				}
			}
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			b, ok := m.list.SelectedItem().(backup)
			if ok {
				backupName := b.raw
				// We want to return to command model
				// We pass m.prevModel, not m
				awaitModel := InitialAwaitModel(m.prevModel, requestRestoreBackup(backupName, m.jwtToken), m.width, m.height, "Restoring backup", "Backup restored!")
				// cmd := requestRestoreBackup(backupName, m.jwtToken)
				cmd := awaitModel.Init()
				return awaitModel, cmd
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case fetchMsg:
		m.list.SetItems(msg.items)
	case taskFinishedMsg:
		return m.prevModel.Update(msg)
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m backupModel) View() string {
	return docStyle.Render(m.list.View())
}

// ///////////////
// HTTP requests
// ///////////////

type fetchMsg struct {
	items []list.Item
}

func requestMakeBackup(jwtToken string) tea.Cmd {
	return func() tea.Msg {
		log.Printf("Enter requestMakeBackup")
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: transport}

		url := fmt.Sprintf(cli.Args.Address("backup"))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("")))
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

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		log.Printf("Return code: %d", resp.StatusCode)

		var msg taskFinishedMsg
		msg.title = "Make Backup"
		msg.msg = fmt.Sprintf("%d %s", resp.StatusCode, "Backup complete")
		msg.sucess = true
		if resp.StatusCode != 200 {
			msg.msg = fmt.Sprintf("%s", body)
			msg.sucess = false
		}

		return msg
	}
}

func requestRestoreBackup(backupName, jwtToken string) tea.Cmd {
	return func() tea.Msg {
		data := map[string]string{"filename": backupName}
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Error marshalling JSON: %v", err)
		}

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: transport}

		// localhost:port/restore
		url := fmt.Sprintf(cli.Args.Address("restore"))
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

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		var msg taskFinishedMsg
		msg.title = "Restore backup"
		msg.msg = fmt.Sprintf("%d %s", resp.StatusCode, "Backup restored")
		msg.sucess = true
		if resp.StatusCode != 200 {
			msg.msg = fmt.Sprintf("%s", body)
			msg.sucess = false
		}

		return msg
	}
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
