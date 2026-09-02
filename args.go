package clic

import (
	"fmt"
	"strings"
)

// Arguments are raw or positional command arguments.
type Arguments []string

func parseArguments(execution *Execution, args Arguments) (Arguments, error) {
	byName, byShort := map[string]*Option{}, map[string]*Option{}
	for _, option := range execution.Options {
		byName[option.Name()] = option
		if option.Meta.ShortName != "" {
			byShort[option.Meta.ShortName] = option
		}
	}
	explicit := map[string]bool{}
	arguments := Arguments{}
	positionalOnly := false
	for index := 0; index < len(args); index++ {
		token := args[index]
		if positionalOnly {
			arguments = append(arguments, token)
			continue
		}
		if token == "--" {
			positionalOnly = true
			continue
		}
		if !strings.HasPrefix(token, "-") || token == "-" {
			arguments = append(arguments, token)
			continue
		}
		name, value, hasValue, option := optionToken(token, byName, byShort)
		if option == nil {
			return nil, fmt.Errorf("unknown option %q", token)
		}
		if explicit[option.Spec.Key] {
			return nil, fmt.Errorf("option %q may only appear once", name)
		}
		explicit[option.Spec.Key] = true
		if option.Spec.Kind == BoolOption {
			if hasValue {
				return nil, fmt.Errorf("boolean option %q does not accept a value", name)
			}
			option.State.Value = true
			option.State.Source = OptionSourceArgument
			continue
		}
		if !hasValue {
			if index+1 == len(args) {
				return nil, fmt.Errorf("option %q requires a value", name)
			}
			index++
			value = args[index]
		}
		parsed, err := parseOptionValue(option, value)
		if err != nil {
			return nil, err
		}
		option.State.Value = parsed
		option.State.Source = OptionSourceArgument
	}
	return arguments, nil
}

func optionToken(token string, names, shorts map[string]*Option) (string, string, bool, *Option) {
	prefix, source := "-", shorts
	if strings.HasPrefix(token, "--") {
		prefix, source = "--", names
	}
	value := strings.TrimPrefix(token, prefix)
	name, supplied, hasValue := strings.Cut(value, "=")
	return prefix + name, supplied, hasValue, source[name]
}
