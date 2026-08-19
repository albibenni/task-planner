package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUsageDescribesCoreCommands(t *testing.T) {
	var output bytes.Buffer
	usage(&output)

	for _, command := range []string{"task-planner config", "task-planner status", "task-planner add", "task-planner delete"} {
		if !strings.Contains(output.String(), command) {
			t.Errorf("usage does not include %q", command)
		}
	}
}
