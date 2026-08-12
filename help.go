package clic

import (
	"fmt"
	"io"
	"strings"
)

// DefaultHelp writes the selected command's visible usage, commands, and
// effective options to Execution.Stdout.
func DefaultHelp(meta CommandMeta, execution *Execution) error {
	if execution == nil || execution.Command == nil {
		return fmt.Errorf("clic help requires a selected command")
	}
	if execution.Stdout == nil {
		return fmt.Errorf("clic help requires stdout")
	}
	var output strings.Builder
	if meta.Description != "" {
		fmt.Fprintf(&output, "%s\n\n", meta.Description)
	}
	output.WriteString("USAGE\n")
	usage := meta.Usage
	if len(usage) == 0 {
		generated := meta.Name
		if len(visibleCommands(execution.Command.Commands)) != 0 {
			generated += " [command]"
		}
		if len(visibleOptions(execution.Options)) != 0 {
			generated += " [options]"
		}
		usage = []string{generated}
	}
	for _, line := range usage {
		fmt.Fprintf(&output, "\t%s\n", line)
	}
	writeCommands(&output, execution.Command.Commands)
	writeOptions(&output, execution.Options)
	_, err := io.WriteString(execution.Stdout, output.String())
	return err
}

func helpOption() *Option {
	return &Option{
		Meta: OptionMeta{
			Name:        "help",
			ShortName:   "h",
			Description: "Show command help.",
			Order:       1000,
		},
		Spec: OptionSpec{
			Key:  "help",
			Kind: BoolOption,
		},
	}
}

func visibleCommands(commands Commands) []*Command {
	visible := make([]*Command, 0, len(commands))
	for _, command := range commands.Sorted() {
		if !command.Meta.Hidden {
			visible = append(visible, command)
		}
	}
	return visible
}

func visibleOptions(options Options) []*Option {
	visible := make([]*Option, 0, len(options))
	for _, option := range options.Sorted() {
		if !option.Meta.Hidden {
			visible = append(visible, option)
		}
	}
	return visible
}

func writeCommands(output *strings.Builder, commands Commands) {
	visible := visibleCommands(commands)
	if len(visible) == 0 {
		return
	}
	output.WriteString("\nAVAILABLE COMMANDS\n\n")
	width := 0
	for _, command := range visible {
		if len(command.Meta.Name) > width {
			width = len(command.Meta.Name)
		}
	}
	for _, command := range visible {
		fmt.Fprintf(output, "\t%-*s  %s\n", width, command.Meta.Name, command.Meta.Description)
	}
}

func writeOptions(output *strings.Builder, options Options) {
	visible := visibleOptions(options)
	if len(visible) == 0 {
		return
	}
	output.WriteString("\nOptions:\n")
	width := 0
	for _, option := range visible {
		if len(optionFlags(*option)) > width {
			width = len(optionFlags(*option))
		}
	}
	for _, option := range visible {
		fmt.Fprintf(output, "\t%-*s  %s\n", width, optionFlags(*option), optionDescription(*option))
	}
}

func optionFlags(option Option) string {
	if option.Meta.ShortName == "" {
		return "    --" + option.Meta.Name
	}
	return "-" + option.Meta.ShortName + ", --" + option.Meta.Name
}

func optionDescription(option Option) string {
	description := option.Meta.Description
	if option.Spec.Required {
		description += " (required)"
	}
	if option.Meta.EnvName != "" {
		description += " (env: " + option.Meta.EnvName + ")"
	}
	if option.Meta.ShowDefault && option.State.Value != nil && fmt.Sprint(option.State.Value) != "" {
		description += fmt.Sprintf(" (default: %v)", option.State.Value)
	}
	return description
}
