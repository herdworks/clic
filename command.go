package clic

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
)

// Command is a reusable command definition and its execution-local state.
type Command struct {
	Meta     CommandMeta
	Spec     CommandSpec
	State    CommandState
	Commands Commands
	Links    Links
}

// Commands maps stable command keys to commands.
type Commands map[string]*Command

// Link redirects one command token to a descendant command path. From is
// resolved relative to the command that owns the link; To contains command
// names or aliases relative to that same command.
type Link struct {
	From string
	To   []string
}

// Links contains a command's undocumented deep links.
type Links []Link

// CommandMeta is host-controlled command presentation. Name defaults to the
// command's Spec.Key when it is empty.
type CommandMeta struct {
	Name        string
	Aliases     []string
	Description string
	Usage       []string
	EnvPrefix   string
	Hidden      bool
	Order       int
}

// MetadataTemplateData supplies the supported values for presentation metadata
// templates. CLI is the root command name and Command is the selected command
// path in its current mount. Name is the literal name of the metadata item
// being rendered. Option templates additionally receive ShortName and EnvName.
type MetadataTemplateData struct {
	CLI       string
	Command   string
	Name      string
	ShortName string
	EnvName   string
}

// CommandSpec is reusable command behavior.
type CommandSpec struct {
	Key       string
	Options   Options
	Help      func(CommandMeta, *Execution) error
	Validator func(*Execution) error
	Handler   func(*Execution) error
}

// CommandState is execution-local command state.
type CommandState struct {
	Arguments Arguments
}

// PositionalArguments returns the number of parsed positional arguments.
func (command *Command) PositionalArguments() int {
	return len(command.State.Arguments)
}

// Name returns the command's presentation name, defaulting to its stable key.
func (command Command) Name() string {
	if command.Meta.Name != "" {
		return command.Meta.Name
	}
	return command.Spec.Key
}

// SubCommand resolves a child by its current mounted name or alias.
func (command Command) SubCommand(name string) (*Command, bool) {
	for _, child := range command.Commands {
		if child.Name() == name {
			return child, true
		}
		for _, alias := range child.Meta.Aliases {
			if alias == name {
				return child, true
			}
		}
	}
	return nil, false
}

// Link resolves a deep link owned by the command.
func (command Command) Link(from string) (*Link, bool) {
	for index := range command.Links {
		if command.Links[index].From == from {
			return &command.Links[index], true
		}
	}
	return nil, false
}

// IsNamespace reports whether a command is an organizational command.
func (command Command) IsNamespace() bool {
	return (len(command.Commands) != 0 || len(command.Links) != 0) && command.Spec.Handler == nil
}

// Help invokes the command's supplied help callback.
func (command Command) Help(execution *Execution) error {
	if command.Spec.Help == nil {
		return ErrHelpNotImplemented
	}
	meta := command.Meta
	meta.Name = command.Name()
	return command.Spec.Help(meta, execution)
}

// Render expands Description and Usage templates with the supplied command
// metadata context. Names and aliases intentionally remain literal because
// clic uses them to validate and resolve command paths.
func (meta CommandMeta) Render(data MetadataTemplateData) (CommandMeta, error) {
	data.Name = meta.Name
	description, err := renderTemplateString(meta.Description, data)
	if err != nil {
		return CommandMeta{}, fmt.Errorf("render command description: %w", err)
	}
	usage := make([]string, len(meta.Usage))
	for index, source := range meta.Usage {
		usage[index], err = renderTemplateString(source, data)
		if err != nil {
			return CommandMeta{}, fmt.Errorf("render command usage %d: %w", index+1, err)
		}
	}
	meta.Description = description
	meta.Usage = usage
	return meta, nil
}

func renderTemplateString(source string, data any) (string, error) {
	template, err := template.New("clic metadata").Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err = template.Execute(&output, data); err != nil {
		return "", err
	}
	return output.String(), nil
}

// ColumnWidth returns the widest column value plus padding.
func (command Command) ColumnWidth(items ColumnListable, field string, padding int) int {
	if padding == 0 {
		padding = 1
	}
	width := 0
	for _, value := range items.ColumnValues(field) {
		if len(value) > width {
			width = len(value)
		}
	}
	return width + padding
}

// Sorted returns commands in presentation order.
func (commands Commands) Sorted() []*Command {
	result := make([]*Command, 0, len(commands))
	for _, command := range commands {
		result = append(result, command)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Meta.Order != result[j].Meta.Order {
			return result[i].Meta.Order < result[j].Meta.Order
		}
		return result[i].Spec.Key < result[j].Spec.Key
	})
	return result
}

// MustCommands composes commands by their stable keys. It panics if a command
// has an empty key or another supplied command uses the same key.
func MustCommands(values ...Command) Commands {
	commands := make(Commands, len(values))
	for _, command := range values {
		if command.Spec.Key == "" {
			panic("clic command has an empty key")
		}
		if _, exists := commands[command.Spec.Key]; exists {
			panic("clic commands have duplicate key " + command.Spec.Key)
		}
		commands[command.Spec.Key] = &command
	}
	return commands
}

// MustCommandsFrom composes commands returned from factories. It panics if a
// factory returns an empty key or a key already produced by another factory.
func MustCommandsFrom(factories ...func() Command) Commands {
	values := make([]Command, len(factories))
	for index, factory := range factories {
		values[index] = factory()
	}
	return MustCommands(values...)
}
