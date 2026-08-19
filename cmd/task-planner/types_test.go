package main

import (
	"encoding/json"
	"testing"
)

func TestProjectJSONFields(t *testing.T) {
	var project project
	if err := json.Unmarshal([]byte(`{"id":"project-id","name":"Inbox"}`), &project); err != nil {
		t.Fatal(err)
	}
	if project.ID != "project-id" || project.Name != "Inbox" {
		t.Fatalf("unexpected project: %#v", project)
	}
}
