package main

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
)

// CLI is a collection of commands
type CLI struct {
	Name        string
	Description string
	Version     string
	Commands    Commands
	Options     CommandOptions // global options
	Args        Arguments
}

func (cli *CLI) Help(err error) {
	usage := cli.Usage()
	usage += cli.Commands.Usage()
	usage += cli.Options.Usage()
	fmt.Println(usage)
	if err != nil {
		fmt.Printf("\nError: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
func (cli *CLI) Usage() (usage string) {
	usage += fmt.Sprintf("%s\n", cli.Description)
	usage += "\nUsage:\n"
	usage += fmt.Sprintf("\t%s", cli.Name)
	if len(cli.Commands) > 0 {
		usage += " [command]"
	}
	if len(cli.Options) > 0 {
		usage += " [options]"
	}
	usage += "\n\n"
	return usage
}

func (cli *CLI) Execute() {
	var err error
	var command *Command
	if len(os.Args) >= 2 {
		command, err = cli.Commands.Lookup(os.Args[1])
	}
	if command != nil {
		command.Options = append(command.Options, cli.Options.Globals()...) // add global options
		command.Execute(os.Args[2:])
	}
	if slices.Contains(os.Args, "-h") || slices.Contains(os.Args, "--help") {
		cli.Help(nil)
	}
	if err != nil {
		cli.Help(err)
	}
	fmt.Printf("%s version %s\n", cli.Name, cli.Version) // noop, print version information and exit
}

// Command is a single action within the CLI.
type Command struct {
	Config      CommandConfig
	Options     CommandOptions
	SubCommands Commands
	Args        Arguments
}
type Commands []*Command

func (cmd *Command) ParseEnvironment() {
	for _, option := range cmd.Options {
		if option.EnvName == "" {
			continue
		} // skip options without defined environment variables
		if value := os.Getenv(option.EnvName); value != "" {
			option.Default = value
		}
	}
}

func (cmd *Command) ParseArguments(args Arguments) (err error) {
	if args.IsEmpty() { return nil } // guard
	flags := cmd.Options.FlagSet()
	if err = flags.Parse(args); err != nil { return err }
	if len(flags.Args()) == 0 { return nil }     // done! (if no remaining arguments)
	cmd.Args = append(cmd.Args, flags.Args()[0]) // argument capture
	return cmd.ParseArguments(flags.Args()[1:])  // RECURSION!! (parse remaining arguments, if any)
}

func (cmd *Command) ProcessArguments(args Arguments) (err error) {
	if cmd.Args.IsEmpty() { return nil } // guard
	if cmd.Config.Helper == nil { return nil }
	err = cmd.Config.Helper(cmd)
	if err != nil { return err }
	return cmd.ParseArguments(args) // reprocess arguments to capture dynamic option values?
}

func (cmd *Command) Validate() (err error) {
	for _, option := range cmd.Options {
		if err := option.Validate(); err != nil { return err }
	}
	if cmd.Config.Validator != nil {
		if err := cmd.Config.Validator(cmd); err != nil { return err }
	}
	return nil
}

func (cmd *Command) Help(err error) {
	usage := cmd.Usage()
	usage += cmd.SubCommands.Usage()
	usage += cmd.Options.Usage()
	fmt.Println(usage)
	if err != nil {
		fmt.Printf("\nError: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func (cmds Commands) Usage() (usage string) {
	if len(cmds) == 0 { return usage }
	if len(cmds) > 0 { usage += "\nAvailable commands:\n" }
	longcmd := slices.MaxFunc(cmds, func(a, b *Command) int {
		return cmp.Compare(len(a.Config.Name), len(b.Config.Name))
	})
	for _, cmd := range cmds {
		usage += fmt.Sprintf("\t%-*s %s\n", len(longcmd.Config.Name)+1, cmd.Config.Name, cmd.Config.Description)
	}
	return usage
}
func (cmd *Command) Usage() (usage string) {
	usage += fmt.Sprintf("%s\n\n", cmd.Config.Description)
	usage += "Usage:\n"
	usage += fmt.Sprintf("\t%s\n", cmd.Config.Usage)
	return usage
}

func (cmd Command) ColumnWidth(items ColumnListable, field string, padding int) (width int) {
	if padding == 0 { padding = 1 }
	values := items.ColumnValues(field)
	for _, v := range values {
		if len(v) > width { width = len(v) }
	}
	return width + padding
}

func (cmds Commands) Lookup(name string) (*Command, error) {
	for _, cmd := range cmds {
		if cmd.Config.Name == name {
			return cmd, nil
		}
	}
	return nil, fmt.Errorf("command \"%s\" not found", name)
}

func (cmd *Command) Execute(args Arguments) {
	// execute subcommand (if requested)
	var subcommand *Command
	if len(cmd.SubCommands) > 0 && len(args) > 0 {
		subcommand, _ = cmd.SubCommands.Lookup(args[0])
	}
	if subcommand != nil {
		subcommand.Options = append(subcommand.Options, cmd.Options.Globals()...) // add global options
		subcommand.Execute(args[1:])
	}

	// parse environment and arguments
	cmd.ParseEnvironment()
	if err := cmd.ParseArguments(args); err != nil { cmd.Help(err) }
	if cmd.Options.Get("help") == "true" { cmd.Help(nil) } // display help if requested
	if err := cmd.Validate(); err != nil { cmd.Help(err) }
	if cmd.Config.Handler == nil {
		cmd.Help(fmt.Errorf("no handler provided for command"))
	}
	if err := cmd.Config.Handler(cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

// CommandConfig holds Command configuration settings.
type CommandConfig struct {
	Name        string
	Description string
	Usage       string
	Helper      func(cmd *Command) error
	Validator   func(cmd *Command) error
	Handler     func(cmd *Command) error
}

// CommandOption holds Command options (e.g. flags).
type CommandOption struct {
	Name        string
	ShortName   string
	EnvName     string
	Description string
	Required    bool
	Secret      bool
	Global      bool
	Value       flag.Value
	Default     string
	Handler     func(string) error

	Boolean    bool
	RawValue   string
	BoolValue  bool
	Registered bool
}
type CommandOptions []*CommandOption

func (option *CommandOption) FlagsNames() string {
	flags := ""
	if option.ShortName != "" {
		flags += fmt.Sprintf("-%s", option.ShortName)
	}
	if len(flags) > 0 {
		flags += ", "
	} else {
		flags = "    "
	}
	if option.Name != "" {
		flags += fmt.Sprintf("--%s", option.Name)
	}
	return flags
}

func (options CommandOptions) FlagColumnWidth() (width int) {
	longflag := slices.MaxFunc(options, func(a, b *CommandOption) int {
		return cmp.Compare(len(a.FlagsNames()), len(b.FlagsNames()))
	})
	return len(longflag.FlagsNames()) + 1
}

func (options CommandOptions) FlagSet() *flag.FlagSet {
	flags := &flag.FlagSet{}
	flags.SetOutput(io.Discard) // suppress flagset output
	flags.Usage = func() {}     // disable flagset usage
	for _, option := range options {
		ParseOptionFlags(flags, option)
	}
	return flags
}

func (option *CommandOption) DefaultValue() (value string) {
	if value := os.Getenv(option.EnvName); value != "" {
		return value
	}
	return fmt.Sprintf("%v", option.Default)
}

func (option *CommandOption) Usage() string {
	usage := option.Description
	if option.Secret {
		return usage
	} // guard against printing secret values in help output
	if option.Required {
		usage += " (required)"
	}
	if option.EnvName != "" {
		usage += fmt.Sprintf(" (env: %s)", option.EnvName)
	}
	if defaultValue := option.DefaultValue(); defaultValue != "" {
		usage += fmt.Sprintf(" (default: %s)", defaultValue)
	}
	return usage
}

func (options CommandOptions) Usage() (usage string) {
	if len(options) == 0 {
		return usage
	}
	usage += "\nOptions:\n"
	for _, option := range options {
		usage += fmt.Sprintf("\t%-*s %s\n", options.FlagColumnWidth(), option.FlagsNames(), option.Usage())
	}
	return usage
}

func (option *CommandOption) Validate() error {
	if option.Required && option.Default == "" && option.RawValue == "" {
		return fmt.Errorf("option \"--%s\" is required", option.Name)
	}
	return nil
}

func (option *CommandOption) String() string {
	if option.Boolean {
		return fmt.Sprintf("%v", option.BoolValue)
	}
	if option.RawValue != "" {
		return option.RawValue
	}
	return option.DefaultValue()
}

func (options CommandOptions) Lookup(name string) *CommandOption {
	for _, option := range options {
		if option.Name == name {
			return option
		}
	}
	return nil
}

func (options CommandOptions) Get(name string) string {
	option := options.Lookup(name)
	if option == nil {
		return ""
	}
	return option.String()
}

func (options CommandOptions) Globals() (globals CommandOptions) {
	for _, option := range options {
		if option.Global {
			globals = append(globals, option)
		}
	}
	return globals
}

// =============================================================================
// Helper types, methods, and functions
// =============================================================================
type Arguments []string

func (args Arguments) IsEmpty() bool {
	return len(args) == 0
}

func ParseOptionFlags(flags *flag.FlagSet, option *CommandOption) {
	if option.Registered {
		return
	}
	ParseOptionFlag(flags, option, option.Name)
	if option.ShortName == "" {
		return
	}
	ParseOptionFlag(flags, option, option.ShortName)
	option.Registered = true
}
func ParseOptionFlag(flags *flag.FlagSet, option *CommandOption, name string) {
	if option.Handler != nil {
		flags.Func(name, option.Description, option.Handler)
		return
	}
	if option.Value != nil {
		flags.Var(option.Value, name, option.Description)
		return
	}
	if option.Boolean {
		flags.BoolVar(&option.BoolValue, name, option.DefaultValue() == "true", option.Description)
		return
	}
	flags.StringVar(&option.RawValue, name, option.DefaultValue(), option.Description)
}

type ColumnListable interface {
	ColumnValues(field string) []string // return the column value for the given name
}

type Comparable interface {
	ComparedBy(field string) string // return the value to compare
}
