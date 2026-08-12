package clic

import "testing"

func TestRunKeepsExecutionInputUnchanged(t *testing.T) {
	inputCommand := &Command{Meta: CommandMeta{Name: "input"}}
	inputOptions := Options{"input": {Spec: OptionSpec{Key: "input"}}}
	execution := Execution{Command: inputCommand, Options: inputOptions}
	cli := commandCLI(nil, func(execution *Execution) error {
		if execution.Command == inputCommand || execution.Options["input"] != nil {
			t.Fatal("handler received caller execution state")
		}
		return nil
	})
	if result := cli.Run(execution, nil); result.Error != nil {
		t.Fatalf("Run() = %#v", result)
	}
	if execution.Command != inputCommand || execution.Options["input"] != inputOptions["input"] {
		t.Fatal("Run() mutated caller execution")
	}
}
