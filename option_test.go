package clic

import "testing"

func TestMustOptionsFrom(t *testing.T) {
	options := MustOptionsFrom(func() Option {
		return Option{Meta: OptionMeta{Name: "name"}, Spec: OptionSpec{Key: "name", Kind: StringOption}}
	})
	if option := options["name"]; option == nil || option.Spec.Key != "name" {
		t.Fatalf("MustOptionsFrom() = %#v", options)
	}
	mustPanic(t, func() {
		MustOptionsFrom(
			func() Option { return Option{Spec: OptionSpec{Key: "same"}} },
			func() Option { return Option{Spec: OptionSpec{Key: "same"}} },
		)
	})
	mustPanic(t, func() { MustOptionsFrom(func() Option { return Option{} }) })
}

func TestOptionNameDefaultsToSpecKey(t *testing.T) {
	var name string
	cli := commandCLI(Options{
		"config": {
			Meta: OptionMeta{
				Description: "Configure {{ .Name }}.",
			},
			Spec: OptionSpec{
				Key:  "config",
				Kind: StringOption,
			},
		},
	}, func(execution *Execution) error {
		name = execution.Options["config"].Meta.Name
		return nil
	})
	if result := cli.Run(testExecution(nil), Arguments{"--config", "site.yaml"}); result.Error != nil {
		t.Fatalf("Run() = %#v", result)
	}
	if name != "config" {
		t.Fatalf("execution option name = %q, want config", name)
	}
	if source := cli.Spec.Options["config"].Meta.Name; source != "" {
		t.Fatalf("source option name = %q, want empty", source)
	}
	if got := cli.Spec.Options["config"].Name(); got != "config" {
		t.Fatalf("Option.Name() = %q, want config", got)
	}
}

func TestOptionsValueGetters(t *testing.T) {
	options := Options{
		"string": {State: OptionState{Value: "value"}},
		"bool":   {State: OptionState{Value: true}},
		"slice":  {State: OptionState{Value: []string{"one", "two"}}},
	}
	if got := options.StringValue("string"); got != "value" {
		t.Fatalf("StringValue() = %q, want value", got)
	}
	if !options.BoolValue("bool") {
		t.Fatal("BoolValue() = false, want true")
	}
	if got, want := options.StringValues("slice"), []string{"one", "two"}; !sameStrings(got, want) {
		t.Fatalf("StringValues() = %q, want %q", got, want)
	}
	for _, key := range []string{"missing", "bool", "slice"} {
		if got := options.StringValue(key); got != "" {
			t.Errorf("StringValue(%q) = %q, want empty", key, got)
		}
	}
	for _, key := range []string{"missing", "string", "slice"} {
		if options.BoolValue(key) {
			t.Errorf("BoolValue(%q) = true, want false", key)
		}
	}
	for _, key := range []string{"missing", "string", "bool"} {
		if got := options.StringValues(key); got != nil {
			t.Errorf("StringValues(%q) = %q, want nil", key, got)
		}
	}
}

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

func TestRunResolvesStringsOptions(t *testing.T) {
	var values [][]string
	cli := commandCLI(Options{
		"tags": {
			Meta: OptionMeta{Name: "tag", ShortName: "t", EnvName: "TAGS", Default: []string{" default ", "", "tag"}},
			Spec: OptionSpec{Key: "tags", Kind: StringsOption},
		},
		"paths": {
			Meta: OptionMeta{Name: "path", EnvName: "PATHS"},
			Spec: OptionSpec{Key: "paths", Kind: StringsOption, Separator: "::"},
		},
	}, func(execution *Execution) error {
		values = append(values, execution.Options.StringValues("tags"), execution.Options.StringValues("paths"))
		return nil
	})
	for _, test := range []struct {
		environment map[string]string
		args        Arguments
		wantTags    []string
		wantPaths   []string
	}{
		{nil, nil, []string{"default", "tag"}, []string{}},
		{map[string]string{"TAGS": " environment ,, tag ", "PATHS": " one ::  :: two "}, nil, []string{"environment", "tag"}, []string{"one", "two"}},
		{map[string]string{"TAGS": "environment", "PATHS": "environment"}, Arguments{"-t", " explicit , , tag ", "--path=one::two"}, []string{"explicit", "tag"}, []string{"one", "two"}},
		{map[string]string{"TAGS": "environment", "PATHS": "environment"}, Arguments{"--tag= , ", "--path=  "}, []string{}, []string{}},
	} {
		if result := cli.Run(testExecution(test.environment), test.args); result != (Result{}) {
			t.Fatalf("Run(%q) = %#v", test.args, result)
		}
		gotTags, gotPaths := values[len(values)-2], values[len(values)-1]
		if !sameStrings(gotTags, test.wantTags) || !sameStrings(gotPaths, test.wantPaths) {
			t.Fatalf("Run(%q) values = %q, %q; want %q, %q", test.args, gotTags, gotPaths, test.wantTags, test.wantPaths)
		}
	}
	if result := cli.Run(testExecution(nil), Arguments{"--tag=one", "--tag=two"}); result.ExitCode != 1 {
		t.Fatalf("repeated list option = %#v", result)
	}
	if source := cli.Spec.Options["tags"].Meta.Default; !sameStrings(source.([]string), []string{" default ", "", "tag"}) {
		t.Fatalf("source default = %q", source)
	}
}

func TestRunValidatesRequiredStringsOptions(t *testing.T) {
	cli := commandCLI(Options{
		"tags": {Meta: OptionMeta{Name: "tag"}, Spec: OptionSpec{Key: "tags", Kind: StringsOption, Required: true}},
	}, func(*Execution) error { return nil })
	for _, args := range []Arguments{nil, {"--tag= , "}} {
		if result := cli.Run(testExecution(nil), args); result.ExitCode != 1 {
			t.Errorf("Run(%q) = %#v, want required-option error", args, result)
		}
	}
	if result := cli.Run(testExecution(nil), Arguments{"--tag=one"}); result != (Result{}) {
		t.Fatalf("Run(nonempty list) = %#v", result)
	}
}

func TestValidateStringsDefaults(t *testing.T) {
	valid := commandCLI(Options{
		"tags": {Meta: OptionMeta{Name: "tag", Default: []string{}}, Spec: OptionSpec{Key: "tags", Kind: StringsOption}},
	}, func(*Execution) error { return nil })
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	invalid := commandCLI(Options{
		"tags": {Meta: OptionMeta{Name: "tag", Default: "one,two"}, Spec: OptionSpec{Key: "tags", Kind: StringsOption}},
	}, func(*Execution) error { return nil })
	if err := invalid.Validate(); err == nil {
		t.Fatal("Validate() succeeded for invalid string-slice default")
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
