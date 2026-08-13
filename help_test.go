package clic

import (
	"bytes"
	"strings"
	"testing"
)

func TestDefaultHelpOrdersEntriesAndIncludesEffectiveOptions(t *testing.T) {
	output := &bytes.Buffer{}
	cli := CLI{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Help: DefaultHelp, Options: Options{
		"config": {Meta: OptionMeta{Name: "config", Global: true, Order: 2}, Spec: OptionSpec{Key: "config", Kind: StringOption}},
	}}, Commands: Commands{
		"second": {Meta: CommandMeta{Name: "second", Order: 2}, Spec: CommandSpec{Key: "second", Handler: func(*Execution) error { return nil }}},
		"first":  {Meta: CommandMeta{Name: "first", Order: 1}, Spec: CommandSpec{Key: "first", Handler: func(*Execution) error { return nil }}},
		"group": {Meta: CommandMeta{Name: "group"}, Spec: CommandSpec{Key: "group", Help: DefaultHelp}, Commands: Commands{
			"child": {Meta: CommandMeta{Name: "child"}, Spec: CommandSpec{Key: "child", Handler: func(*Execution) error { return nil }}},
		}},
	}}}
	execution := testExecution(nil)
	execution.Stdout = output
	if result := cli.Run(execution, nil); result.Error != nil {
		t.Fatalf("root help Run() = %#v", result)
	}
	if strings.Index(output.String(), "first") > strings.Index(output.String(), "second") {
		t.Fatalf("root help order = %q", output.String())
	}
	output.Reset()
	if result := cli.Run(execution, Arguments{"group"}); result.Error != nil {
		t.Fatalf("group help Run() = %#v", result)
	}
	if !strings.Contains(output.String(), "--config") {
		t.Fatalf("group help omitted inherited option: %q", output.String())
	}
}

func TestRunHelpOptionCallsCommandHelpBeforeValidationOrHandling(t *testing.T) {
	var helpCalls, validatorCalls, handlerCalls int
	cli := commandCLI(Options{
		"config": {Meta: OptionMeta{Name: "config"}, Spec: OptionSpec{Key: "config", Kind: StringOption, Required: true}},
	}, func(*Execution) error {
		handlerCalls++
		return nil
	})
	cli.Spec.Help = func(meta CommandMeta, execution *Execution) error {
		helpCalls++
		if meta.Name != "tool" || !execution.Options["help"].BoolValue() {
			t.Fatalf("help execution = %#v", execution)
		}
		return nil
	}
	cli.Spec.Validator = func(*Execution) error {
		validatorCalls++
		return nil
	}
	for _, args := range []Arguments{{"--help"}, {"-h"}} {
		if result := cli.Run(testExecution(nil), args); result != (Result{}) {
			t.Fatalf("Run(%q) = %#v", args, result)
		}
	}
	if helpCalls != 2 || validatorCalls != 0 || handlerCalls != 0 {
		t.Fatalf("help %d, validator %d, handler %d", helpCalls, validatorCalls, handlerCalls)
	}
	withoutHelp := commandCLI(nil, func(*Execution) error { return nil })
	if result := withoutHelp.Run(testExecution(nil), Arguments{"--help"}); result.ExitCode != 1 {
		t.Fatalf("Run() without help callback = %#v", result)
	}
}

func TestDefaultHelpListsImplicitHelpOptionLast(t *testing.T) {
	output := &bytes.Buffer{}
	cli := commandCLI(Options{
		"config": {Meta: OptionMeta{Name: "config"}, Spec: OptionSpec{Key: "config", Kind: StringOption}},
	}, func(*Execution) error { return nil })
	cli.Spec.Help = DefaultHelp
	execution := testExecution(nil)
	execution.Stdout = output
	if result := cli.Run(execution, Arguments{"--help"}); result.Error != nil {
		t.Fatalf("Run() = %#v", result)
	}
	if strings.Index(output.String(), "--config") > strings.Index(output.String(), "--help") {
		t.Fatalf("help option order = %q", output.String())
	}
}

func TestDefaultHelpAlignsLongOnlyOptions(t *testing.T) {
	output := &bytes.Buffer{}
	execution := &Execution{
		Stdout:  output,
		Command: &Command{Meta: CommandMeta{Name: "tool"}},
		Options: Options{
			"config":    {Meta: OptionMeta{Name: "config", ShortName: "c", Description: "Configuration."}, Spec: OptionSpec{Key: "config", Kind: StringOption}},
			"providers": {Meta: OptionMeta{Name: "providers", Description: "Provider names."}, Spec: OptionSpec{Key: "providers", Kind: StringOption}},
		},
	}
	if err := DefaultHelp(CommandMeta{Name: "tool"}, execution); err != nil {
		t.Fatalf("DefaultHelp() = %v", err)
	}
	if !strings.Contains(output.String(), "\t    --providers  Provider names.\n") {
		t.Fatalf("help output = %q", output.String())
	}
}

func TestDefaultHelpShowsNonEmptyEffectiveOptionValues(t *testing.T) {
	output := &bytes.Buffer{}
	execution := &Execution{
		Stdout:  output,
		Command: &Command{Meta: CommandMeta{Name: "tool"}},
		Options: Options{
			"effective": {Meta: OptionMeta{Name: "effective", Default: "metadata", ShowDefault: true}, Spec: OptionSpec{Key: "effective", Kind: StringOption}, State: OptionState{Value: "environment"}},
			"empty":     {Meta: OptionMeta{Name: "empty", Default: "metadata", ShowDefault: true}, Spec: OptionSpec{Key: "empty", Kind: StringOption}, State: OptionState{Value: ""}},
		},
	}
	if err := DefaultHelp(CommandMeta{Name: "tool"}, execution); err != nil {
		t.Fatalf("DefaultHelp() = %v", err)
	}
	if !strings.Contains(output.String(), "(default: environment)") {
		t.Fatalf("help output = %q", output.String())
	}
	if strings.Contains(output.String(), "(default: metadata)") {
		t.Fatalf("empty effective value shown: %q", output.String())
	}
}

func TestDefaultHelpUsesDefaultOptionNameInTemplates(t *testing.T) {
	output := &bytes.Buffer{}
	execution := &Execution{
		Stdout: output,
		Command: &Command{
			Spec: CommandSpec{
				Key: "tool",
			},
		},
		Options: Options{
			"config": {
				Meta: OptionMeta{
					Description: "Configure {{ .Name }}.",
				},
				Spec: OptionSpec{
					Key:  "config",
					Kind: StringOption,
				},
			},
		},
	}
	if err := DefaultHelp(CommandMeta{}, execution); err != nil {
		t.Fatalf("DefaultHelp() = %v", err)
	}
	if !strings.Contains(output.String(), "--config  Configure config.\n") {
		t.Fatalf("help output = %q", output.String())
	}
}

func TestDefaultHelpRendersMetadataTemplatesForMountedCommands(t *testing.T) {
	output := &bytes.Buffer{}
	cli := CLI{
		Command: Command{
			Meta: CommandMeta{
				Name:        "host",
				Description: "{{ .CLI }} at {{ .Command }}.",
			},
			Spec: CommandSpec{
				Key:  "host",
				Help: DefaultHelp,
			},
			Commands: Commands{
				"theme": {
					Meta: CommandMeta{
						Name:        "theme",
						Description: "Manage {{ .Name }} at {{ .Command }}.",
						Usage: []string{
							"{{ .Name }}: {{ .Command }} [command]",
						},
					},
					Spec: CommandSpec{
						Key:  "theme",
						Help: DefaultHelp,
						Options: Options{
							"config": {
								Meta: OptionMeta{
									Name:        "config",
									ShortName:   "c",
									EnvName:     "HOST_CONFIG",
									Description: "Configure {{ .Name }} (-{{ .ShortName }}) from {{ .EnvName }} for {{ .CLI }} at {{ .Command }}.",
								},
								Spec: OptionSpec{
									Key:  "config",
									Kind: StringOption,
								},
							},
						},
					},
				},
			},
		},
	}
	execution := testExecution(nil)
	execution.Stdout = output
	if result := cli.Run(execution, Arguments{"theme", "--help"}); result.Error != nil {
		t.Fatalf("Run() = %#v", result)
	}
	for _, want := range []string{
		"Manage theme at host theme.",
		"theme: host theme [command]",
		"Configure config (-c) from HOST_CONFIG for host at host theme.",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("help output missing %q:\n%s", want, output.String())
		}
	}
}
