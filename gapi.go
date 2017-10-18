package gapi

import (
	"fmt"
	"net/http"
)

type CommandFunc func(c *http.Client, argv []string) int
type CommandEntry struct {
	Func  CommandFunc
	Scope string
}

var (
	commands = make(map[string]*CommandEntry)
)

func ListCommandNames() []string {
	ret := make([]string, 0, len(commands))
	for k, _ := range commands {
		ret = append(ret, k)
	}
	return ret
}

func GetCommand(name string) (*CommandEntry, error) {
	if c, ok := commands[name]; !ok {
		return nil, fmt.Errorf("*** Command %s not exist.", name)
	} else {
		return c, nil
	}
}

func RegisterCommand(name, scope string, main CommandFunc) {
	if _, ok := commands[name]; ok {
		panic(name + " already registered")
	}
	commands[name] = &CommandEntry{
		Func:  main,
		Scope: scope,
	}
}
