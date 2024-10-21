package cli

import (
	"fmt"
)

const (
	PORT_MIN = 1
	PORT_MAX = 65535
)

var Args CliArgs

type CliArgs struct {
	Host string `short:"a" name:"host" default:"localhost" help:"Host"`
	Port int    `short:"p" name:"port" help:"Port" required:""`
}

func (a CliArgs) Validate() error {
	if a.Port < PORT_MIN || a.Port > PORT_MAX {
		return fmt.Errorf("Port out of range")
	}
	return nil
}

func (a CliArgs) Address(path string) string {
	return fmt.Sprintf("https://%s:%d/%s", a.Host, a.Port, path)
}
