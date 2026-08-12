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
