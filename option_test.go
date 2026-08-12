package clic

import "testing"

func TestValidateEffectiveOptionCollisions(t *testing.T) {
	tests := []struct {
		name string
		cli  CLI
		want bool
	}{
		{"semantic key", commandTree(option("shared", "parent", "p", true), option("shared", "child", "c", false)), false},
		{"long name", commandTree(option("parent", "same", "p", true), option("child", "same", "c", false)), false},
		{"short name", commandTree(option("parent", "parent", "s", true), option("child", "child", "s", false)), false},
		{"non-global parent option is not inherited", commandTree(option("shared", "parent", "p", false), option("shared", "child", "c", false)), true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.cli.Validate(); (err == nil) != test.want {
				t.Fatalf("Validate() = %v, want valid %t", err, test.want)
			}
		})
	}
}

func TestValidateRejectsOptionsThatCollideWithImplicitHelp(t *testing.T) {
	tests := []Option{
		option("help", "other", "o", false),
		option("other", "help", "o", false),
		option("other", "other", "h", false),
	}
	for _, option := range tests {
		cli := commandCLI(Options{option.Spec.Key: &option}, func(*Execution) error { return nil })
		cli.Spec.Help = DefaultHelp
		if err := cli.Validate(); err == nil {
			t.Errorf("Validate() succeeded for option %#v", option)
		}
	}
}

func TestRunRejectsInvalidOptionArguments(t *testing.T) {
	cli := commandCLI(Options{
		"value":   {Meta: OptionMeta{Name: "value", ShortName: "v"}, Spec: OptionSpec{Key: "value", Kind: StringOption, Required: true}},
		"verbose": {Meta: OptionMeta{Name: "verbose", ShortName: "q"}, Spec: OptionSpec{Key: "verbose", Kind: BoolOption}},
	}, func(*Execution) error { return nil })
	for _, args := range []Arguments{{"--unknown"}, {"--value", "one", "--value", "two"}, {"--value", "one", "-v", "two"}, {"--value", "one", "--verbose", "-q"}, {"--value="}} {
		if result := cli.Run(testExecution(nil), args); result.ExitCode != 1 {
			t.Errorf("Run(%q) = %#v, want framework warning", args, result)
		}
	}
}

func TestRunResolvesOptionValuesInPrecedenceOrder(t *testing.T) {
	var values []string
	cli := commandCLI(Options{
		"string": {Meta: OptionMeta{Name: "string", EnvName: "STRING", Default: "default"}, Spec: OptionSpec{Key: "string", Kind: StringOption}},
		"zero":   {Meta: OptionMeta{Name: "zero"}, Spec: OptionSpec{Key: "zero", Kind: StringOption}},
		"bool":   {Meta: OptionMeta{Name: "bool", EnvName: "BOOL", Default: false}, Spec: OptionSpec{Key: "bool", Kind: BoolOption}},
	}, func(execution *Execution) error {
		values = append(values, execution.Options["string"].StringValue()+":"+execution.Options["zero"].StringValue()+":"+boolString(execution.Options["bool"].BoolValue()))
		return nil
	})
	for _, test := range []struct {
		environment map[string]string
		args        Arguments
	}{{}, {map[string]string{"STRING": "environment", "BOOL": "true"}, nil}, {map[string]string{"STRING": "", "BOOL": "false"}, nil}, {map[string]string{"STRING": "environment", "BOOL": "false"}, Arguments{"--string=", "--bool"}}} {
		if result := cli.Run(testExecution(test.environment), test.args); result != (Result{}) {
			t.Fatalf("Run(%q) = %#v", test.args, result)
		}
	}
	if got, want := values, []string{"default::false", "environment::true", "::false", "::true"}; !sameStrings(got, want) {
		t.Fatalf("values = %q, want %q", got, want)
	}
}

func commandTree(parent, child Option) CLI {
	return CLI{Command: Command{Meta: CommandMeta{Name: "tool"}, Spec: CommandSpec{Key: "tool", Options: Options{parent.Spec.Key: &parent}}, Commands: Commands{
		"child": {Meta: CommandMeta{Name: "child"}, Spec: CommandSpec{Key: "child", Options: Options{child.Spec.Key: &child}, Handler: func(*Execution) error { return nil }}},
	}}}
}

func option(key, name, short string, global bool) Option {
	return Option{Meta: OptionMeta{Name: name, ShortName: short, Global: global}, Spec: OptionSpec{Key: key, Kind: StringOption}}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
