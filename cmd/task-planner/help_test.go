package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpDescribesCommandGroupsAndLeafCommands(t *testing.T) {
	for _, test := range []struct {
		path     []string
		expected []string
	}{
		{[]string{"auth"}, []string{"login", "logout", "projects"}},
		{[]string{"config", "default-project"}, []string{"set", "clear", "Todoist project ID"}},
		{[]string{"delete"}, []string{"two characters", "Supabase", "database only"}},
		{[]string{"delete", "old"}, []string{"past", "bulk deletion", "Supabase only", "Todoist tasks stay unchanged"}},
		{[]string{"completion", "bash"}, []string{"task-planner completion bash", "Bash"}},
	} {
		var output bytes.Buffer
		if err := commandHelp(&output, test.path); err != nil {
			t.Fatalf("help %s: %v", strings.Join(test.path, " "), err)
		}
		for _, expected := range test.expected {
			if !strings.Contains(output.String(), expected) {
				t.Errorf("help %s lacks %q: %s", strings.Join(test.path, " "), expected, output.String())
			}
		}
	}
}

func TestHelpRejectsUnknownCommandPath(t *testing.T) {
	var output bytes.Buffer
	if err := commandHelp(&output, []string{"auth", "missing"}); err == nil || !strings.Contains(err.Error(), "unknown help topic") {
		t.Fatalf("unknown help topic should explain the error: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("unknown help topic should not show unrelated help: %s", output.String())
	}
}

func TestHelpCoversEveryCommandPath(t *testing.T) {
	paths := [][]string{
		{"config"}, {"config", "default-project"}, {"config", "default-project", "set"}, {"config", "default-project", "clear"},
		{"auth"}, {"auth", "login"}, {"auth", "logout"}, {"auth", "projects"},
		{"status"}, {"check"}, {"add"}, {"list"}, {"plans"}, {"delete"}, {"delete", "old"},
		{"completion"}, {"completion", "bash"}, {"completion", "zsh"}, {"help"},
	}
	for _, path := range paths {
		var output bytes.Buffer
		if err := commandHelp(&output, path); err != nil || !strings.Contains(output.String(), "Usage:") {
			t.Errorf("help %s should be available: %v, %s", strings.Join(path, " "), err, output.String())
		}
	}
}
