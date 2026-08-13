package clic

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestRunResolvesCommandsOptionsAndKeepsStateLocal(t *testing.T) {
	var received []string
	var verbose []bool
	cli := CLI{Command: Command{
		Meta: CommandMeta{Name: "tool"},
		Spec: CommandSpec{Key: "tool", Options: Options{
			"verbose": {Meta: OptionMeta{Name: "verbose", ShortName: "v", Global: true}, Spec: OptionSpec{Key: "verbose", Kind: BoolOption}},
		}},
		Commands: Commands{"run": {
			Meta: CommandMeta{Name: "run", Aliases: []string{"r"}},
			Spec: CommandSpec{Key: "run", Options: Options{
				"title": {Meta: OptionMeta{Name: "title", ShortName: "t", EnvName: "TITLE", Default: "default"}, Spec: OptionSpec{Key: "title", Kind: StringOption, Required: true}},
			}, Handler: func(execution *Execution) error {
				received = append(received, execution.Command.State.Arguments[0], execution.Options["title"].StringValue())
				verbose = append(verbose, execution.Options["verbose"].BoolValue())
				return nil
			}},
		}},
	}}
	execution := testExecution(nil)
	direct := copyOptions(cli.Spec.Options)
	directExecution := Execution{Options: direct}
	if _, err := parseArguments(&directExecution, Arguments{"-v"}); err != nil || !direct["verbose"].BoolValue() {
		t.Fatalf("direct parse = %#v, %#v", err, direct["verbose"])
	}
	result := cli.Run(execution, Arguments{"r", "first", "--verbose", "--title=Dr."})
	if result != (Result{}) {
		t.Fatalf("Run() = %#v", result)
	}
	if got, want := received, []string{"first", "Dr."}; !sameStrings(got, want) {
		t.Fatalf("received = %q, want %q", got, want)
	}
	if got, want := verbose, []bool{true}; !sameBools(got, want) {
		t.Fatalf("verbose = %t, want %t", got, want)
	}
	if cli.Commands["run"].State.Arguments != nil || cli.Commands["run"].Spec.Options["title"].State.Value != nil {
		t.Fatal("canonical command state was mutated")
	}
	result = cli.Run(testExecution(nil), Arguments{"run", "second", "--title", "Prof."})
	if result != (Result{}) {
		t.Fatalf("second Run() = %#v", result)
	}
	if got, want := received, []string{"first", "Dr.", "second", "Prof."}; !sameStrings(got, want) {
		t.Fatalf("received = %q, want %q", got, want)
	}
	if got, want := verbose, []bool{true, false}; !sameBools(got, want) {
		t.Fatalf("verbose = %t, want %t", got, want)
	}
}

func TestCommandNameDefaultsToSpecKey(t *testing.T) {
	var name, path string
	cli := CLI{
		Command: Command{
			Spec: CommandSpec{
				Key: "tool",
				Help: func(meta CommandMeta, execution *Execution) error {
					name = meta.Name
					path = execution.CommandPathString()
					return nil
				},
			},
			Commands: Commands{
				"run": {
					Spec: CommandSpec{
						Key: "run",
						Handler: func(execution *Execution) error {
							name = execution.Command.Meta.Name
							path = execution.CommandPathString()
							return nil
						},
					},
				},
			},
		},
	}
	if result := cli.Run(testExecution(nil), Arguments{"run"}); result.Error != nil {
		t.Fatalf("Run(run) = %#v", result)
	}
	if name != "run" || path != "tool run" {
		t.Fatalf("run command = name %q, path %q", name, path)
	}
	if result := cli.Run(testExecution(nil), nil); result.Error != nil {
		t.Fatalf("Run() = %#v", result)
	}
	if name != "tool" || path != "tool" {
		t.Fatalf("root command = name %q, path %q", name, path)
	}
}

func TestRunResolvesCommandLinksWithoutDisplayingThemInHelp(t *testing.T) {
	var path string
	output := &bytes.Buffer{}
	cli := CLI{
		Command: Command{
			Meta: CommandMeta{
				Name: "hyperctl",
			},
			Spec: CommandSpec{
				Key:  "hyperctl",
				Help: DefaultHelp,
			},
			Commands: Commands{
				"htxt": {
					Meta: CommandMeta{
						Name: "website",
					},
					Spec: CommandSpec{
						Key: "htxt",
					},
					Commands: Commands{
						"build": {
							Meta: CommandMeta{
								Name: "build",
							},
							Spec: CommandSpec{
								Key: "build",
								Handler: func(execution *Execution) error {
									path = execution.CommandPathString()
									return nil
								},
							},
						},
					},
				},
			},
			Links: Links{
				{
					From: "build",
					To: []string{
						"website",
						"build",
					},
				},
			},
		},
	}
	if result := cli.Run(testExecution(nil), Arguments{"build"}); result.Error != nil {
		t.Fatalf("Run(build) = %#v", result)
	}
	if path != "hyperctl website build" {
		t.Fatalf("command path = %q, want hyperctl website build", path)
	}
	execution := testExecution(nil)
	execution.Stdout = output
	if result := cli.Run(execution, Arguments{"--help"}); result.Error != nil {
		t.Fatalf("Run(--help) = %#v", result)
	}
	if strings.Contains(output.String(), "build") {
		t.Fatalf("root help includes link: %q", output.String())
	}
}

func TestRunOptionParsingAndPrecedence(t *testing.T) {
	var got string
	cli := commandCLI(Options{
		"string": {Meta: OptionMeta{Name: "string", ShortName: "s", EnvName: "STRING", Default: "default"}, Spec: OptionSpec{Key: "string", Kind: StringOption}},
		"bool":   {Meta: OptionMeta{Name: "bool", ShortName: "b", EnvName: "BOOL", Default: false}, Spec: OptionSpec{Key: "bool", Kind: BoolOption}},
	}, func(execution *Execution) error {
		got = execution.Options["string"].StringValue()
		if execution.Options["bool"].BoolValue() {
			got += ":true"
		} else {
			got += ":false"
		}
		return nil
	})
	result := cli.Run(testExecution(map[string]string{"STRING": "environment", "BOOL": "false"}), Arguments{"before", "-s", "explicit", "--bool", "after"})
	if result != (Result{}) || got != "explicit:true" {
		t.Fatalf("Run() = %#v, got %q", result, got)
	}
	result = cli.Run(testExecution(map[string]string{"STRING": "", "BOOL": "false"}), Arguments{"--bool"})
	if result != (Result{}) || got != ":true" {
		t.Fatalf("bare boolean = %#v, %q", result, got)
	}
	result = cli.Run(testExecution(map[string]string{"STRING": "environment", "BOOL": "true"}), nil)
	if result != (Result{}) || got != "environment:true" {
		t.Fatalf("boolean environment = %#v, %q", result, got)
	}
	result = cli.Run(testExecution(map[string]string{"BOOL": "not-a-bool"}), nil)
	if result.ExitCode != 1 {
		t.Fatalf("invalid boolean environment = %#v", result)
	}
	result = cli.Run(testExecution(nil), Arguments{"--bool=false"})
	if result.ExitCode != 1 {
		t.Fatalf("boolean value = %#v", result)
	}
}

func TestRunResolvesEnvironmentPrefixesByOptionDeclaration(t *testing.T) {
	var config, theme, palette string
	cli := CLI{
		Command: Command{
			Meta: CommandMeta{
				Name:      "tool",
				EnvPrefix: "ROOT_",
			},
			Spec: CommandSpec{
				Key: "tool",
				Options: Options{
					"config": {
						Meta: OptionMeta{
							Name:    "config",
							EnvName: "{{ .EnvPrefix }}CONFIG",
							Global:  true,
						},
						Spec: OptionSpec{
							Key:  "config",
							Kind: StringOption,
						},
					},
				},
			},
			Commands: Commands{
				"theme": {
					Meta: CommandMeta{
						Name:      "theme",
						EnvPrefix: "THEME_",
					},
					Spec: CommandSpec{
						Key: "theme",
						Options: Options{
							"theme": {
								Meta: OptionMeta{
									Name:    "theme",
									EnvName: "{{ .EnvPrefix }}THEME",
									Global:  true,
								},
								Spec: OptionSpec{
									Key:  "theme",
									Kind: StringOption,
								},
							},
						},
					},
					Commands: Commands{
						"child": {
							Meta: CommandMeta{
								Name: "child",
							},
							Spec: CommandSpec{
								Key: "child",
								Options: Options{
									"palette": {
										Meta: OptionMeta{
											Name:    "palette",
											EnvName: "{{ .EnvPrefix }}PALETTE",
										},
										Spec: OptionSpec{
											Key:  "palette",
											Kind: StringOption,
										},
									},
								},
								Handler: func(execution *Execution) error {
									config = execution.Options["config"].StringValue()
									theme = execution.Options["theme"].StringValue()
									palette = execution.Options["palette"].StringValue()
									return nil
								},
							},
						},
					},
				},
			},
		},
	}
	result := cli.Run(testExecution(map[string]string{
		"ROOT_CONFIG":   "site.yaml",
		"THEME_THEME":   "theme.json",
		"THEME_PALETTE": "warm",
	}), Arguments{"theme", "child"})
	if result.Error != nil {
		t.Fatalf("Run() = %#v", result)
	}
	if config != "site.yaml" || theme != "theme.json" || palette != "warm" {
		t.Fatalf("environment values = config %q, theme %q, palette %q", config, theme, palette)
	}
}

func TestRunReportsFrameworkAndHandlerFailures(t *testing.T) {
	namespace := CLI{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Help: func(CommandMeta, *Execution) error { return nil }}, Commands: Commands{"group": {Meta: CommandMeta{Name: "group"}, Spec: CommandSpec{Key: "group"}, Commands: Commands{"child": {Meta: CommandMeta{Name: "child"}, Spec: CommandSpec{Key: "child"}}}}}}}
	if result := namespace.Run(testExecution(nil), Arguments{"group", "unknown"}); result.ExitCode != 1 {
		t.Fatalf("unknown namespace command = %#v", result)
	}
	if result := namespace.Run(testExecution(nil), Arguments{"group"}); !errors.Is(result.Error, ErrHelpNotImplemented) || result.ExitCode != 1 {
		t.Fatalf("namespace help = %#v", result)
	}
	bad := commandCLI(nil, func(*Execution) error { return errors.New("handler failure") })
	if result := bad.Run(testExecution(nil), nil); result.ExitCode != 2 {
		t.Fatalf("handler failure = %#v", result)
	}
}

func TestRunDispatchesCommandsBeforeHandlers(t *testing.T) {
	var calls []string
	cli := CLI{Command: Command{
		Meta: CommandMeta{Name: "tool"},
		Spec: CommandSpec{Key: "tool", Handler: func(execution *Execution) error {
			calls = append(calls, "root:"+strings.Join(execution.Command.State.Arguments, ","))
			return nil
		}},
		Commands: Commands{
			"group": {Meta: CommandMeta{Name: "group", Aliases: []string{"g"}}, Spec: CommandSpec{Key: "group", Handler: func(*Execution) error {
				calls = append(calls, "group")
				return nil
			}}, Commands: Commands{
				"child": {Meta: CommandMeta{Name: "child", Aliases: []string{"c"}}, Spec: CommandSpec{Key: "child", Handler: func(*Execution) error {
					calls = append(calls, "child")
					return nil
				}}},
			}},
		},
	}}
	if result := cli.Run(testExecution(nil), Arguments{"g", "c"}); result != (Result{}) {
		t.Fatalf("nested alias Run() = %#v", result)
	}
	if result := cli.Run(testExecution(nil), Arguments{"plain"}); result != (Result{}) {
		t.Fatalf("unmatched positional Run() = %#v", result)
	}
	if got, want := calls, []string{"child", "root:plain"}; !sameStrings(got, want) {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestRunCallsEntrypointHelpForAnEmptyNamespace(t *testing.T) {
	var received CommandMeta
	cli := CLI{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Help: func(meta CommandMeta, execution *Execution) error {
		received = meta
		return nil
	}}, Commands: Commands{
		"child": {Meta: CommandMeta{Name: "child"}, Spec: CommandSpec{Key: "child", Handler: func(*Execution) error { return nil }}},
	}}}
	if result := cli.Run(testExecution(nil), nil); result != (Result{}) {
		t.Fatalf("Run() = %#v", result)
	}
	if received.Name != "tool" {
		t.Fatalf("help metadata = %#v", received)
	}
}

func TestRunLifecycleDoesNotWriteFrameworkErrors(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	validatorCalled, handlerCalled := false, false
	cli := commandCLI(Options{
		"value": {Meta: OptionMeta{Name: "value", Default: "default"}, Spec: OptionSpec{Key: "value", Kind: StringOption}},
	}, func(*Execution) error {
		handlerCalled = true
		return nil
	})
	cli.Spec.Validator = func(execution *Execution) error {
		validatorCalled = true
		if execution.Options["value"].StringValue() != "resolved" {
			return errors.New("validator ran before option resolution")
		}
		return errors.New("validator failure")
	}
	execution := testExecution(map[string]string{"VALUE": "resolved"})
	execution.Stdout, execution.Stderr = stdout, stderr
	cli.Spec.Options["value"].Meta.EnvName = "VALUE"
	if result := cli.Run(execution, nil); result.ExitCode != 1 || !validatorCalled || handlerCalled {
		t.Fatalf("validator Run() = %#v, validator %t, handler %t", result, validatorCalled, handlerCalled)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("framework failure wrote stdout %q stderr %q", stdout.String(), stderr.String())
	}
	if result := cli.Run(execution, Arguments{"--unknown"}); result.ExitCode != 1 {
		t.Fatalf("parser Run() = %#v", result)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("parser failure wrote stdout %q stderr %q", stdout.String(), stderr.String())
	}
	namespace := CLI{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool"}, Commands: Commands{
		"child": {Meta: CommandMeta{Name: "child"}, Spec: CommandSpec{Key: "child", Handler: func(*Execution) error { return nil }}},
	}}}
	if result := namespace.Run(execution, nil); !errors.Is(result.Error, ErrHelpNotImplemented) || result.ExitCode != 1 {
		t.Fatalf("help Run() = %#v", result)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("help failure wrote stdout %q stderr %q", stdout.String(), stderr.String())
	}
}

func TestValidateRejectsInvalidComposition(t *testing.T) {
	tests := []CLI{
		{Command: Command{Meta: CommandMeta{Name: "tool"}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool"}, Commands: Commands{"child": {Meta: CommandMeta{Name: "child", Aliases: []string{""}}, Spec: CommandSpec{Key: "child"}}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{"wrong": {Meta: OptionMeta{Name: "option"}, Spec: OptionSpec{Key: "option", Kind: StringOption}}}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{"option": nil}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{"option": {Meta: OptionMeta{Name: "option"}, Spec: OptionSpec{Key: "option", Kind: OptionKind(99)}}}}}},
	}
	for index, cli := range tests {
		if err := cli.Validate(); err == nil {
			t.Errorf("test %d: Validate() succeeded", index)
		}
	}
}

func TestValidateRejectsCompositionCollisions(t *testing.T) {
	tests := []CLI{
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool"}, Commands: Commands{"wrong": {Meta: CommandMeta{Name: "child"}, Spec: CommandSpec{Key: "child"}}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool"}, Commands: Commands{"one": {Meta: CommandMeta{Name: "same"}, Spec: CommandSpec{Key: "one"}}, "two": {Meta: CommandMeta{Name: "two", Aliases: []string{"same"}}, Spec: CommandSpec{Key: "two"}}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{"flag": {Meta: OptionMeta{Name: "flag", Default: true}, Spec: OptionSpec{Key: "flag", Kind: StringOption}}}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{"flag": {Meta: OptionMeta{Name: "flag"}, Spec: OptionSpec{Key: "flag", Kind: BoolOption, Required: true}}}}}},
		{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{"flag": {Meta: OptionMeta{Name: "flag", Default: true}, Spec: OptionSpec{Key: "flag", Kind: BoolOption}}}}}},
	}
	for index, cli := range tests {
		if err := cli.Validate(); err == nil {
			t.Errorf("test %d: Validate() succeeded", index)
		}
	}
}

func TestSortedAndConcurrentRuns(t *testing.T) {
	commands := Commands{"b": {Meta: CommandMeta{Name: "b", Order: 1}, Spec: CommandSpec{Key: "b"}}, "a": {Meta: CommandMeta{Name: "a", Order: 1}, Spec: CommandSpec{Key: "a"}}}
	if got := commands.Sorted(); got[0].Spec.Key != "a" {
		t.Fatalf("command order = %q", got[0].Spec.Key)
	}
	options := Options{"b": {Meta: OptionMeta{Name: "b", Order: 1}, Spec: OptionSpec{Key: "b"}}, "a": {Meta: OptionMeta{Name: "a", Order: 1}, Spec: OptionSpec{Key: "a"}}}
	if got := options.Sorted(); got[0].Spec.Key != "a" {
		t.Fatalf("option order = %q", got[0].Spec.Key)
	}
	cli := commandCLI(Options{"value": {Meta: OptionMeta{Name: "value"}, Spec: OptionSpec{Key: "value", Kind: StringOption}}}, func(*Execution) error { return nil })
	var group sync.WaitGroup
	for range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			if result := cli.Run(testExecution(nil), Arguments{"--value", "x"}); result.Error != nil {
				t.Error(result.Error)
			}
		}()
	}
	group.Wait()
	if cli.Spec.Options["value"].State.Value != nil {
		t.Fatal("concurrent runs mutated canonical option")
	}
}

func TestDefaultHelp(t *testing.T) {
	output := &bytes.Buffer{}
	command := &Command{
		Meta: CommandMeta{Name: "tool"},
		Commands: Commands{
			"visible": {Meta: CommandMeta{Name: "visible", Description: "Visible command."}, Spec: CommandSpec{Key: "visible"}},
			"hidden":  {Meta: CommandMeta{Name: "hidden", Hidden: true}, Spec: CommandSpec{Key: "hidden"}},
		},
	}
	execution := &Execution{
		Stdout:  output,
		Command: command,
		Options: Options{
			"config": {Meta: OptionMeta{Name: "config", ShortName: "c", Description: "Configuration.", EnvName: "CONFIG", Default: "site.yaml", ShowDefault: true}, Spec: OptionSpec{Key: "config", Kind: StringOption, Required: true}, State: OptionState{Value: "site.yaml"}},
			"hidden": {Meta: OptionMeta{Name: "hidden", Hidden: true}, Spec: OptionSpec{Key: "hidden", Kind: BoolOption}},
		},
	}
	if err := DefaultHelp(CommandMeta{Name: "tool", Description: "A test tool."}, execution); err != nil {
		t.Fatalf("DefaultHelp() = %v", err)
	}
	want := "A test tool.\n\nUSAGE\n\ttool [command] [options]\n\nAVAILABLE COMMANDS\n\n\tvisible  Visible command.\n\nOPTIONS\n\t-c, --config  Configuration. (required) (env: CONFIG) (default: site.yaml)\n"
	if got := output.String(); got != want {
		t.Fatalf("DefaultHelp() = %q, want %q", got, want)
	}
	output.Reset()
	if err := DefaultHelp(CommandMeta{Name: "tool", Usage: []string{"tool first", "tool second"}}, execution); err != nil {
		t.Fatalf("DefaultHelp() = %v", err)
	}
	want = "USAGE\n\ttool first\n\ttool second\n\nAVAILABLE COMMANDS\n\n\tvisible  Visible command.\n\nOPTIONS\n\t-c, --config  Configuration. (required) (env: CONFIG) (default: site.yaml)\n"
	if got := output.String(); got != want {
		t.Fatalf("DefaultHelp() multiple usage = %q, want %q", got, want)
	}
}

func commandCLI(options Options, handler func(*Execution) error) CLI {
	return CLI{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: options, Handler: handler}}}
}

func testExecution(environment map[string]string) Execution {
	return Execution{Context: context.Background(), Stdout: &bytes.Buffer{}, LookupEnv: func(key string) (string, bool) { value, ok := environment[key]; return value, ok }}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameBools(left, right []bool) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func mustPanic(t *testing.T, callback func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	callback()
}
