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
	Host          string `short:"a" name:"host" default:"localhost" help:"Host"`
	Port          int    `short:"p" name:"port" help:"Port" required:""`
	TimeOffsetMin int    `short:"t" name:"time-offset" help:"Time offset used to diplay the backup time" default:"0"`
}

func (a CliArgs) Validate() error {
	if a.Port == 0 {
		return fmt.Errorf("you must specify a port")
	}
	if a.Port < PORT_MIN || a.Port > PORT_MAX {
		return fmt.Errorf("port out of range")
	}
	return nil
}

func (a CliArgs) Address(path string) string {
	return fmt.Sprintf("https://%s:%d/%s", a.Host, a.Port, path)
}
