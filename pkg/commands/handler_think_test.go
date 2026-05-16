package commands

import (
	"fmt"
	"testing"
)

func TestSwitchThinkLevel_Success(t *testing.T) {
	rt := &Runtime{
		SwitchThinkLevel: func(level string) (string, error) {
			return "off", nil
		},
	}
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), rt)

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think to high",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	want := "Switched thinking from off to high"
	if reply != want {
		t.Fatalf("reply=%q, want=%q", reply, want)
	}
}

func TestSwitchThinkLevel_InvalidLevel(t *testing.T) {
	rt := &Runtime{
		SwitchThinkLevel: func(level string) (string, error) {
			return "", fmt.Errorf("unexpected call")
		},
	}
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), rt)

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think to invalid",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply == "Switched thinking" {
		t.Fatal("expected error for invalid level")
	}
}

func TestSwitchThinkLevel_MissingToKeyword(t *testing.T) {
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think high",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply == "" {
		t.Fatal("expected usage message")
	}
}

func TestSwitchThinkLevel_MissingValue(t *testing.T) {
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think to",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply == "" {
		t.Fatal("expected usage message")
	}
}

func TestSwitchThinkLevel_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think to high",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("reply=%q, want unavailable message", reply)
	}
}

func TestSwitchThinkLevel_Error(t *testing.T) {
	rt := &Runtime{
		SwitchThinkLevel: func(level string) (string, error) {
			return "", fmt.Errorf("provider does not support thinking")
		},
	}
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), rt)

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think to high",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	want := "provider does not support thinking"
	if reply != want {
		t.Fatalf("reply=%q, want=%q", reply, want)
	}
}

func TestThink_WeirdWhitespace(t *testing.T) {
	rt := &Runtime{
		SwitchThinkLevel: func(level string) (string, error) {
			if level != "high" {
				return "", fmt.Errorf("expected 'high', got %q", level)
			}
			return "off", nil
		},
	}
	ex := NewExecutor(NewRegistryWithDefs(BuiltinProvider{}.CommandDefinitions()), rt)

	var reply string
	res := ex.Execute(nil, Request{
		Text: "/think to HIGH",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	want := "Switched thinking from off to high"
	if reply != want {
		t.Fatalf("reply=%q, want=%q", reply, want)
	}
}

func TestIsValidThinkLevel(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"off", true},
		{"low", true},
		{"medium", true},
		{"high", true},
		{"xhigh", true},
		{"adaptive", true},
		{"", false},
		{"invalid", false},
		{"OFF", false},  // case-sensitive
		{"High", false}, // case-sensitive
	}
	for _, tc := range cases {
		got := isValidThinkLevel(tc.input)
		if got != tc.want {
			t.Errorf("isValidThinkLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
