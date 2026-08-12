package clic

import (
	"context"
	"io"
)

// Execution is the complete injected runtime boundary for one CLI run.
type Execution struct {
	Context          context.Context
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	LookupEnv        func(string) (string, bool)
	WorkingDirectory string
	Command          *Command
	Options          Options
}

// Option returns an effective option by its stable key.
func (execution *Execution) Option(key string) (*Option, bool) {
	option, ok := execution.Options[key]
	return option, ok
}
