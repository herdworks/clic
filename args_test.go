package clic

import "testing"

func TestRunTreatsArgumentsAfterDoubleDashAsPositional(t *testing.T) {
	var received Arguments
	cli := commandCLI(Options{
		"value": {Meta: OptionMeta{Name: "value"}, Spec: OptionSpec{Key: "value", Kind: StringOption}},
	}, func(execution *Execution) error {
		received = execution.Command.State.Arguments
		return nil
	})
	result := cli.Run(testExecution(nil), Arguments{"--value", "first", "--", "-dash", "--unknown"})
	if result != (Result{}) {
		t.Fatalf("Run() = %#v", result)
	}
	want := Arguments{"-dash", "--unknown"}
	if got := received; !sameStrings(got, want) {
		t.Fatalf("arguments = %q, want %q", got, want)
	}
}
