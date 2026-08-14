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
	if err := validateCommandTree(cli.Command, "entrypoint"); err != nil {
		return err
	}
	return validateOptionTree(cli.Command, nil)
}

// Run validates, resolves, parses, and executes one command without process-global side effects.
func (cli CLI) Run(execution Execution, args Arguments) Result {
	var err error

	// does the CLI contain a valid command tree?
	if err = cli.Validate(); err != nil {
		return WarningResult(err)
	}

	// resolve the execution command, options, and arguments
	execution.Command, execution.Options, execution.CommandPath, args, err = cli.resolveCommand(args)
	if err != nil {
		return WarningResult(err)
	}
	if execution.Command.State.Arguments, err = parseArguments(&execution, args); err != nil {
		return WarningResult(err)
	}
	if execution.Command.IsNamespace() && len(execution.Command.State.Arguments) != 0 {
		return WarningResult(fmt.Errorf("unknown command %q", execution.Command.State.Arguments[0]))
	}
	if err := resolveOptionValues(execution.Options, &execution); err != nil {
		return WarningResult(err)
	}
	if help, ok := execution.Options["help"]; ok && help.BoolValue() {
		if err := execution.Command.Help(&execution); err != nil {
			return WarningResult(err)
		}
		return Result{}
	}
	if err := validateRequiredOptions(execution.Options); err != nil {
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
		return WarningResult(fmt.Errorf("command %q has no handler", execution.Command.Name()))
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

func validateCommandTree(command Command, location string) error {
	if command.Spec.Key == "" {
		return fmt.Errorf("%s command key is required", location)
	}
	if err := validateOptions(command.Spec.Options, command.Spec.Key); err != nil {
		return err
	}
	mounted := map[string]bool{}
	for key, child := range command.Commands {
		if child == nil {
			return fmt.Errorf("command %q contains a nil child", command.Spec.Key)
		}
		if key != child.Spec.Key {
			return fmt.Errorf("command map key %q does not match command key %q", key, child.Spec.Key)
		}
		if err := validateCommandTree(*child, command.Spec.Key+"."+key); err != nil {
			return err
		}
		mounted[key] = true
	}
	names := map[string]string{}
	for _, child := range command.Commands {
		for _, name := range append([]string{child.Name()}, child.Meta.Aliases...) {
			if name == "" {
				return fmt.Errorf("command %q has an empty alias", child.Spec.Key)
			}
			if other, exists := names[name]; exists {
				return fmt.Errorf("commands %q and %q share name or alias %q under %q", other, child.Spec.Key, name, command.Spec.Key)
			}
			names[name] = child.Spec.Key
			mounted[name] = true
		}
	}
	links := map[string]bool{}
	for _, link := range command.Links {
		if link.From == "" {
			return fmt.Errorf("command %q contains a link with an empty from", command.Spec.Key)
		}
		if mounted[link.From] {
			return fmt.Errorf("command %q link from %q conflicts with a mounted command", command.Spec.Key, link.From)
		}
		if links[link.From] {
			return fmt.Errorf("command %q contains duplicate link from %q", command.Spec.Key, link.From)
		}
		links[link.From] = true
		if len(link.To) == 0 {
			return fmt.Errorf("command %q link from %q has an empty destination", command.Spec.Key, link.From)
		}
		target := &command
		for _, name := range link.To {
			child, ok := target.SubCommand(name)
			if !ok {
				return fmt.Errorf("command %q link from %q targets unknown command %q", command.Spec.Key, link.From, name)
			}
			target = child
		}
	}
	return nil
}

func validateOptionTree(command Command, globals []*Option) error {
	effective := append([]*Option(nil), globals...)
	for _, option := range command.Spec.Options {
		effective = append(effective, option)
	}
	if command.Spec.Help != nil {
		effective = append(effective, helpOption())
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
		if err := validateOptionTree(*child, nextGlobals); err != nil {
			return err
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
		if option.Spec.Key == "" {
			return fmt.Errorf("option key is required")
		}
		if option.Spec.Kind != StringOption && option.Spec.Kind != BoolOption && option.Spec.Kind != StringsOption {
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
		if names[option.Name()] {
			return fmt.Errorf("effective options for command %q duplicate name --%s", commandKey, option.Name())
		}
		names[option.Name()] = true
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
	case StringsOption:
		_, ok := value.([]string)
		return ok
	default:
		return false
	}
}

func (cli CLI) resolveCommand(args Arguments) (*Command, Options, []string, Arguments, error) {
	selected := &cli.Command
	ancestors := []*Command{}
	commandPath := []string{selected.Name()}
	args = append(Arguments(nil), args...)
	for len(args) > 0 && args[0] != "--" && !strings.HasPrefix(args[0], "-") {
		child, ok := selected.SubCommand(args[0])
		if ok {
			ancestors = append(ancestors, selected)
			selected = child
			commandPath = append(commandPath, selected.Name())
			args = args[1:]
			continue
		}
		link, ok := selected.Link(args[0])
		if !ok {
			break
		}
		for _, name := range link.To {
			child, ok := selected.SubCommand(name)
			if !ok {
				return nil, nil, nil, nil, fmt.Errorf("command %q link from %q targets unknown command %q", selected.Spec.Key, link.From, name)
			}
			ancestors = append(ancestors, selected)
			selected = child
			commandPath = append(commandPath, selected.Name())
		}
		args = args[1:]
	}
	command := *selected
	command.Meta.Name = selected.Name()
	command.Meta.Aliases = append([]string(nil), selected.Meta.Aliases...)
	command.State = CommandState{}
	effective := Options{}
	commands := append(ancestors, selected)
	envPrefix := ""
	for index, source := range commands {
		if source.Meta.EnvPrefix != "" {
			envPrefix = source.Meta.EnvPrefix
		}
		for key, option := range source.Spec.Options {
			if index != len(commands)-1 && !option.Meta.Global {
				continue
			}
			copy, err := copyOptionWithEnvPrefix(option, envPrefix)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			effective[key] = copy
		}
	}
	command.Spec.Options = copyOptions(selected.Spec.Options)
	if selected.Spec.Help != nil {
		effective["help"] = helpOption()
	}
	return &command, effective, commandPath, args, nil
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
	copy.Meta.Name = option.Name()
	copy.State = OptionState{}
	return &copy
}

func copyOptionWithEnvPrefix(option *Option, envPrefix string) (*Option, error) {
	copy := copyOption(option)
	meta, err := copy.Meta.ResolveEnvName(envPrefix)
	if err != nil {
		return nil, err
	}
	copy.Meta = meta
	return copy, nil
}

func resolveOptionValues(options Options, execution *Execution) error {
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
				if option.Spec.Kind == StringsOption {
					option.State.Value = normalizeStringSlice(option.Meta.Default.([]string))
				} else {
					option.State.Value = option.Meta.Default
				}
			}
			if option.State.Value == nil {
				if option.Spec.Kind == BoolOption {
					option.State.Value = false
				} else if option.Spec.Kind == StringsOption {
					option.State.Value = []string{}
				} else {
					option.State.Value = ""
				}
			}
		}
	}
	return nil
}

func validateRequiredOptions(options Options) error {
	for _, option := range options {
		if option.Spec.Required {
			if option.Spec.Kind == StringsOption && len(option.StringSliceValue()) == 0 {
				return fmt.Errorf("option --%s is required", option.Name())
			}
			if option.Spec.Kind != StringsOption && option.StringValue() == "" {
				return fmt.Errorf("option --%s is required", option.Name())
			}
		}
	}
	return nil
}

func parseOptionValue(option *Option, value string) (any, error) {
	if option.Spec.Kind == StringOption {
		return value, nil
	}
	if option.Spec.Kind == StringsOption {
		return normalizeStringSlice(strings.Split(value, option.separator())), nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid boolean environment value %q for option --%s", value, option.Name())
	}
	return parsed, nil
}
