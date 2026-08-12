package clic

import "testing"

func TestMustCommandsFrom(t *testing.T) {
	commands := MustCommandsFrom(func() Command {
		return Command{Meta: CommandMeta{Name: "run"}, Spec: CommandSpec{Key: "run"}}
	})
	if command := commands["run"]; command == nil || command.Spec.Key != "run" {
		t.Fatalf("MustCommandsFrom() = %#v", commands)
	}
	mustPanic(t, func() {
		MustCommandsFrom(
			func() Command { return Command{Spec: CommandSpec{Key: "same"}} },
			func() Command { return Command{Spec: CommandSpec{Key: "same"}} },
		)
	})
	mustPanic(t, func() { MustCommandsFrom(func() Command { return Command{} }) })
}

func TestCommandPositionalArguments(t *testing.T) {
	command := &Command{State: CommandState{Arguments: Arguments{"one", "two"}}}
	if got := command.PositionalArguments(); got != 2 {
		t.Fatalf("PositionalArguments() = %d, want 2", got)
	}
}
