package main

import (
	"fmt"
	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"log"
	"mctui/app"
	"mctui/cli"
	"os"
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

	var err error

	_ = kong.Parse(&cli.Args)
	err = cli.Args.Validate()
	if err != nil {
		panic(err.Error())
	}

	// program := tea.NewProgram(app.InitialLoginModel())
	// program := tea.NewProgram(app.InitialLoginModel(), tea.WithMouseCellMotion(), tea.WithAltScreen())
	program := tea.NewProgram(app.InitialAwaitModel(10, 10, "wait the timer", "done"), tea.WithMouseCellMotion(), tea.WithAltScreen())
	if err != nil {

		fmt.Printf("Uh oh, there was an error: %v\n", err)
	}
	program.Run()

}
