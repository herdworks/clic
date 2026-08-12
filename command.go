package clic

import "sort"

// Command is a reusable command definition and its execution-local state.
type Command struct {
	Meta     CommandMeta
	Spec     CommandSpec
	State    CommandState
	Commands Commands
}

// Commands maps stable command keys to commands.
type Commands map[string]*Command

// CommandMeta is host-controlled command presentation.
type CommandMeta struct {
	Name        string
	Aliases     []string
	Description string
	Usage       []string
	Hidden      bool
	Order       int
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

// SubCommand resolves a child by its current mounted name or alias.
func (command Command) SubCommand(name string) (*Command, bool) {
	for _, child := range command.Commands {
		if child.Meta.Name == name {
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

// IsNamespace reports whether a command is an organizational command.
func (command Command) IsNamespace() bool {
	return len(command.Commands) != 0 && command.Spec.Handler == nil
}

// Help invokes the command's supplied help callback.
func (command Command) Help(execution *Execution) error {
	if command.Spec.Help == nil {
		return ErrHelpNotImplemented
	}
	return command.Spec.Help(command.Meta, execution)
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

// MustCommandsFrom composes commands returned from factories. It panics if a
// factory returns an empty key or a key already produced by another factory.
func MustCommandsFrom(factories ...func() Command) Commands {
	commands := make(Commands, len(factories))
	for _, factory := range factories {
		command := factory()
		if command.Spec.Key == "" {
			panic("clic command factory returned an empty key")
		}
		if _, exists := commands[command.Spec.Key]; exists {
			panic("clic command factories returned duplicate key " + command.Spec.Key)
		}
		commands[command.Spec.Key] = &command
	}
	return commands
}
