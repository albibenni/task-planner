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
		if !strings.Contains(script, "status") {
			t.Errorf("%s completion lacks status command", shell)
		}
	}
}

func TestCompletionScriptRejectsUnknownShell(t *testing.T) {
	if _, err := completionScript("fish"); err == nil {
		t.Fatal("expected an error for fish")
	}
}
