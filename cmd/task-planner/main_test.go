package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUsageDescribesCoreCommands(t *testing.T) {
	var output bytes.Buffer
	usage(&output)

	for _, command := range []string{"task-planner config", "task-planner config default-project set", "task-planner status", "task-planner check", "task-planner add", "task-planner list", "task-planner delete", "task-planner delete old"} {
		if !strings.Contains(output.String(), command) {
			t.Errorf("usage does not include %q", command)
		}
	}
}
