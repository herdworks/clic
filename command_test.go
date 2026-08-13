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

func TestValidateLinks(t *testing.T) {
	validTarget := Commands{
		"htxt": {
			Meta: CommandMeta{
				Name:    "website",
				Aliases: []string{"cms"},
			},
			Spec: CommandSpec{
				Key: "htxt",
			},
			Commands: Commands{
				"build": {
					Spec: CommandSpec{
						Key: "build",
					},
				},
			},
		},
	}
	tests := []struct {
		name  string
		links Links
		valid bool
	}{
		{
			name: "valid",
			links: Links{
				{
					From: "legacy-build",
					To: []string{
						"website",
						"build",
					},
				},
			},
			valid: true,
		},
		{
			name: "target alias",
			links: Links{
				{
					From: "legacy-cms-build",
					To: []string{
						"cms",
						"build",
					},
				},
			},
			valid: true,
		},
		{
			name: "empty from",
			links: Links{
				{
					To: []string{"website"},
				},
			},
		},
		{
			name: "duplicate from",
			links: Links{
				{
					From: "legacy-build",
					To:   []string{"website"},
				},
				{
					From: "legacy-build",
					To:   []string{"website"},
				},
			},
		},
		{
			name: "mounted command key",
			links: Links{
				{
					From: "htxt",
					To:   []string{"website"},
				},
			},
		},
		{
			name: "empty target",
			links: Links{
				{
					From: "legacy-build",
				},
			},
		},
		{
			name: "unknown target",
			links: Links{
				{
					From: "legacy-build",
					To:   []string{"missing"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cli := CLI{
				Command: Command{
					Spec: CommandSpec{
						Key: "tool",
					},
					Commands: validTarget,
					Links:    test.links,
				},
			}
			if err := cli.Validate(); (err == nil) != test.valid {
				t.Fatalf("Validate() = %v, want valid %t", err, test.valid)
			}
		})
	}
}
