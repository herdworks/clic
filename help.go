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
	if meta.Name == "" {
		meta.Name = execution.Command.Name()
	}
	data := execution.MetadataTemplateData()
	if data.CLI == "" {
		data.CLI = meta.Name
	}
	if data.Command == "" {
		data.Command = meta.Name
	}
	meta, err := meta.Render(data)
	if err != nil {
		return err
	}
	var output strings.Builder
	if meta.Description != "" {
		fmt.Fprintf(&output, "%s\n\n", meta.Description)
	}
	output.WriteString("USAGE\n")
	usage := meta.Usage
	if len(usage) == 0 {
		generated := data.Command
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
	if err := writeCommands(&output, execution.Command.Commands, data); err != nil {
		return err
	}
	if err := writeOptions(&output, execution.Options, data); err != nil {
		return err
	}
	_, err = io.WriteString(execution.Stdout, output.String())
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

func writeCommands(output *strings.Builder, commands Commands, parent MetadataTemplateData) error {
	visible := visibleCommands(commands)
	if len(visible) == 0 {
		return nil
	}
	metas := make([]CommandMeta, len(visible))
	for index, command := range visible {
		data := parent
		data.Command = strings.TrimSpace(parent.Command + " " + command.Name())
		meta := command.Meta
		meta.Name = command.Name()
		meta, err := meta.Render(data)
		if err != nil {
			return err
		}
		metas[index] = meta
	}
	output.WriteString("\nAVAILABLE COMMANDS\n\n")
	width := 0
	for _, meta := range metas {
		if len(meta.Name) > width {
			width = len(meta.Name)
		}
	}
	for _, meta := range metas {
		fmt.Fprintf(output, "\t%-*s  %s\n", width, meta.Name, meta.Description)
	}
	return nil
}

func writeOptions(output *strings.Builder, options Options, data MetadataTemplateData) error {
	visible := visibleOptions(options)
	if len(visible) == 0 {
		return nil
	}
	metas := make([]OptionMeta, len(visible))
	for index, option := range visible {
		meta := option.Meta
		meta.Name = option.Name()
		meta, err := meta.Render(data)
		if err != nil {
			return err
		}
		metas[index] = meta
	}
	output.WriteString("\nOPTIONS\n")
	width := 0
	for index, meta := range metas {
		option := *visible[index]
		option.Meta = meta
		if len(optionFlags(option)) > width {
			width = len(optionFlags(option))
		}
	}
	for index, meta := range metas {
		option := *visible[index]
		option.Meta = meta
		fmt.Fprintf(output, "\t%-*s  %s\n", width, optionFlags(option), optionDescription(option))
	}
	return nil
}

func optionFlags(option Option) string {
	if option.Meta.ShortName == "" {
		return "    --" + option.Name()
	}
	return "-" + option.Meta.ShortName + ", --" + option.Name()
}

func optionDescription(option Option) string {
	description := option.Meta.Description
	if option.Spec.Required {
		description += " (required)"
	}
	if option.Meta.EnvName != "" {
		description += " (env: " + option.Meta.EnvName + ")"
	}
	if option.Meta.ShowDefault && option.State.Value != nil && optionValueString(option) != "" {
		description += fmt.Sprintf(" (default: %s)", optionValueString(option))
	}
	return description
}

func optionValueString(option Option) string {
	if option.Spec.Kind == StringsOption {
		return strings.Join(option.StringSliceValue(), option.separator())
	}
	return fmt.Sprint(option.State.Value)
}
