// Package clic provides primitives for composing command-line interfaces.
package clic

import (
	"fmt"
	"strconv"
	"strings"
)

// CLI is a composed command-line interface entrypoint.
type CLI struct {
	Command
	Version string
}

// Result is the outcome of one CLI run.
type Result struct {
	ExitCode int
	Error    error
}

// Validate verifies that a CLI composition has an unambiguous command and option namespace.
func (cli CLI) Validate() error {
	if err := validateCommand(cli.Command, "entrypoint"); err != nil {
		return err
	}
	return validateTree(cli.Command, nil)
}

// Run validates, resolves, parses, and executes one command without process-global side effects.
func (cli CLI) Run(execution Execution, args Arguments) Result {
	var err error

	// does the CLI contain a valid command tree?
	if err = cli.Validate(); err != nil {
		return WarningResult(err)
	}

	// resolve the execution command, options, and arguments
	execution.Command, execution.Options, args = cli.resolveCommand(args)
	if execution.Command.State.Arguments, err = parseArguments(&execution, args); err != nil {
		return WarningResult(err)
	}
	if execution.Command.IsNamespace() && len(execution.Command.State.Arguments) != 0 {
		return WarningResult(fmt.Errorf("unknown command %q", execution.Command.State.Arguments[0]))
	}
	if err := resolveOptions(execution.Options, &execution); err != nil {
		return WarningResult(err)
	}

	// validate command
	if execution.Command.Spec.Validator != nil {
		if err := execution.Command.Spec.Validator(&execution); err != nil {
			return WarningResult(err)
		}
	}

	// generate help output for namespace commands
	if execution.Command.IsNamespace() {
		if err := execution.Command.Help(&execution); err != nil {
			return WarningResult(err)
		}
		return Result{}
	}

	// execute command
	if execution.Command.Spec.Handler == nil {
		return WarningResult(fmt.Errorf("command %q has no handler", execution.Command.Meta.Name))
	} else if err := execution.Command.Spec.Handler(&execution); err != nil {
		return Result{ExitCode: 2, Error: err}
	} else {
		return Result{}
	}
}

func WarningResult(err error) Result {
	return Result{
		ExitCode: 1,
		Error:    err,
	}
}

func validateCommand(command Command, location string) error {
	if command.Spec.Key == "" {
		return fmt.Errorf("%s command key is required", location)
	}
	if command.Meta.Name == "" {
		return fmt.Errorf("command %q name is required", command.Spec.Key)
	}
	if err := validateOptions(command.Spec.Options, command.Spec.Key); err != nil {
		return err
	}
	for key, child := range command.Commands {
		if child == nil {
			return fmt.Errorf("command %q contains a nil child", command.Spec.Key)
		}
		if key != child.Spec.Key {
			return fmt.Errorf("command map key %q does not match command key %q", key, child.Spec.Key)
		}
		if err := validateCommand(*child, command.Spec.Key+"."+key); err != nil {
			return err
		}
	}
	return validateCommandNames(command.Commands, command.Spec.Key)
}

func validateTree(command Command, globals []*Option) error {
	effective := append([]*Option(nil), globals...)
	for _, option := range command.Spec.Options {
		effective = append(effective, option)
	}
	if err := validateEffectiveOptions(effective, command.Spec.Key); err != nil {
		return err
	}
	nextGlobals := append([]*Option(nil), globals...)
	for _, option := range command.Spec.Options {
		if option.Meta.Global {
			nextGlobals = append(nextGlobals, option)
		}
	}
	for _, child := range command.Commands {
		if err := validateTree(*child, nextGlobals); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandNames(commands Commands, parent string) error {
	names := map[string]string{}
	for _, command := range commands {
		for _, name := range append([]string{command.Meta.Name}, command.Meta.Aliases...) {
			if name == "" {
				return fmt.Errorf("command %q has an empty alias", command.Spec.Key)
			}
			if other, exists := names[name]; exists {
				return fmt.Errorf("commands %q and %q share name or alias %q under %q", other, command.Spec.Key, name, parent)
			}
			names[name] = command.Spec.Key
		}
	}
	return nil
}

func validateOptions(options Options, commandKey string) error {
	for key, option := range options {
		if option == nil {
			return fmt.Errorf("command %q contains a nil option", commandKey)
		}
		if key != option.Spec.Key {
			return fmt.Errorf("option map key %q does not match option key %q", key, option.Spec.Key)
		}
		if option.Spec.Key == "" || option.Meta.Name == "" {
			return fmt.Errorf("option key and name are required")
		}
		if option.Spec.Kind != StringOption && option.Spec.Kind != BoolOption {
			return fmt.Errorf("option %q has unsupported kind", option.Spec.Key)
		}
		if option.Spec.Required && option.Spec.Kind == BoolOption {
			return fmt.Errorf("boolean option %q cannot be required", option.Spec.Key)
		}
		if option.Meta.Default != nil && !defaultMatches(option.Spec.Kind, option.Meta.Default) {
			return fmt.Errorf("option %q default does not match its kind", option.Spec.Key)
		}
		if option.Spec.Kind == BoolOption && option.Meta.Default == true {
			return fmt.Errorf("boolean option %q default must be false", option.Spec.Key)
		}
	}
	return nil
}

func validateEffectiveOptions(options []*Option, commandKey string) error {
	keys, names, shorts := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, option := range options {
		if keys[option.Spec.Key] {
			return fmt.Errorf("effective options for command %q duplicate key %q", commandKey, option.Spec.Key)
		}
		keys[option.Spec.Key] = true
		if names[option.Meta.Name] {
			return fmt.Errorf("effective options for command %q duplicate name --%s", commandKey, option.Meta.Name)
		}
		names[option.Meta.Name] = true
		if option.Meta.ShortName != "" {
			if shorts[option.Meta.ShortName] {
				return fmt.Errorf("effective options for command %q duplicate short name -%s", commandKey, option.Meta.ShortName)
			}
			shorts[option.Meta.ShortName] = true
		}
	}
	return nil
}

func defaultMatches(kind OptionKind, value any) bool {
	switch kind {
	case StringOption:
		_, ok := value.(string)
		return ok
	case BoolOption:
		_, ok := value.(bool)
		return ok
	default:
		return false
	}
}

func (cli CLI) resolveCommand(args Arguments) (*Command, Options, Arguments) {
	selected := &cli.Command
	ancestors := []*Command{}
	args = append(Arguments(nil), args...)
	for len(args) > 0 && args[0] != "--" && !strings.HasPrefix(args[0], "-") {
		child, ok := selected.SubCommand(args[0])
		if !ok {
			break
		}
		ancestors = append(ancestors, selected)
		selected = child
		args = args[1:]
	}
	command := *selected
	command.Meta.Aliases = append([]string(nil), selected.Meta.Aliases...)
	command.State = CommandState{}
	command.Spec.Options = copyOptions(selected.Spec.Options)
	effective := Options{}
	for _, ancestor := range ancestors {
		for key, option := range ancestor.Spec.Options {
			if option.Meta.Global {
				effective[key] = copyOption(option)
			}
		}
	}
	for key, option := range selected.Spec.Options {
		effective[key] = copyOption(option)
	}
	return &command, effective, args
}

func copyOptions(options Options) Options {
	result := Options{}
	for key, option := range options {
		result[key] = copyOption(option)
	}
	return result
}

func copyOption(option *Option) *Option {
	copy := *option
	copy.State = OptionState{}
	return &copy
}

func resolveOptions(options Options, execution *Execution) error {
	for _, option := range options {
		if option.State.Value == nil {
			if option.Meta.EnvName != "" && execution.LookupEnv != nil {
				if value, ok := execution.LookupEnv(option.Meta.EnvName); ok {
					parsed, err := parseOptionValue(option, value)
					if err != nil {
						return err
					}
					option.State.Value = parsed
				}
			}
			if option.State.Value == nil && option.Meta.Default != nil {
				option.State.Value = option.Meta.Default
			}
			if option.State.Value == nil {
				if option.Spec.Kind == BoolOption {
					option.State.Value = false
				} else {
					option.State.Value = ""
				}
			}
		}
		if option.Spec.Required && option.StringValue() == "" {
			return fmt.Errorf("option --%s is required", option.Meta.Name)
		}
	}
	return nil
}

func parseOptionValue(option *Option, value string) (any, error) {
	if option.Spec.Kind == StringOption {
		return value, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid boolean environment value %q for option --%s", value, option.Meta.Name)
	}
	return parsed, nil
}
