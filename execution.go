package clic

import (
	"context"
	"io"
	"strings"
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
	CommandPath      []string
	Options          Options
}

// RootCommandName returns the current CLI entrypoint's mounted name.
func (execution *Execution) RootCommandName() string {
	if len(execution.CommandPath) == 0 {
		return ""
	}
	return execution.CommandPath[0]
}

// CommandPathString returns the selected command's canonical mounted path.
func (execution *Execution) CommandPathString() string {
	return strings.Join(execution.CommandPath, " ")
}

// MetadataTemplateData returns the selected command's presentation metadata
// context. It is available to custom help renderers as well as DefaultHelp.
func (execution *Execution) MetadataTemplateData() MetadataTemplateData {
	return MetadataTemplateData{
		CLI:     execution.RootCommandName(),
		Command: execution.CommandPathString(),
	}
}

// Option returns an effective option by its stable key.
func (execution *Execution) Option(key string) (*Option, bool) {
	option, ok := execution.Options[key]
	return option, ok
}
