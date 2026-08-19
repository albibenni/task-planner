package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateTodoistTaskUsesMockAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/tasks" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing Todoist token: %q", request.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["content"] != "Plan the day" || payload["due_date"] != "2026-08-20" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"todoist-123"}`))
	}))
	defer server.Close()

	originalURL := todoistAPIBaseURL
	todoistAPIBaseURL = server.URL
	t.Cleanup(func() { todoistAPIBaseURL = originalURL })
	t.Setenv("TODOIST_API_TOKEN", "test-token")
	p := plan{ID: "plan-the-day", Content: "Plan the day", ProjectID: "project-123"}
	dueDate := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)

	id, err := createTodoistTask(p, dueDate)
	if err != nil || id != "todoist-123" {
		t.Fatalf("got ID %q, err %v", id, err)
	}
}
