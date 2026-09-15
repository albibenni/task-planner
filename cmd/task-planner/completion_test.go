package main

import (
	"strings"
	"testing"
)

func TestCompletionScripts(t *testing.T) {
	for shell, expected := range map[string]string{"bash": "complete -F _task_planner task-planner", "zsh": "compdef _task_planner task-planner"} {
		script, err := completionScript(shell)
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		if !strings.Contains(script, expected) {
			t.Errorf("%s completion lacks %q", shell, expected)
		}
		if !strings.Contains(script, "status") || !strings.Contains(script, "check") {
			t.Errorf("%s completion lacks status or check command", shell)
		}
		if !strings.Contains(script, "default-project") || !strings.Contains(script, "set clear") {
			t.Errorf("%s completion lacks default project commands", shell)
		}
		if !strings.Contains(script, "help") || !strings.Contains(script, "login logout projects") {
			t.Errorf("%s completion lacks command help topics", shell)
		}
	}
}

func TestCompletionScriptRejectsUnknownShell(t *testing.T) {
	if _, err := completionScript("fish"); err == nil {
		t.Fatal("expected an error for fish")
	}
}
