package main

import (
	"fmt"
	"io"
	"log"
	"mctui/app"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
		log.SetOutput(f)
	} else {
		log.SetOutput(io.Discard)
	}

	// var cli CLI
	// _ = kong.Parse(&cli)
	// address := fmt.Sprintf("http://%s", cli.Address)

	var err error
	// program := tea.NewProgram(app.InitialLoginModel())
	program := tea.NewProgram(app.InitialCommandModel())
	if err != nil {

		fmt.Printf("Uh oh, there was an error: %v\n", err)
	}
	program.Run()

}
