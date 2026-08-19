package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestSharedPostgresSchedules(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	os.Setenv("SUPABASE_DB_URL", url)
	if err := databaseReachable(url); err != nil {
		t.Fatalf("database should be reachable: %v", err)
	}
	if err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "truncate task_planner_schedule_tasks, task_planner_schedules")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	p := plan{ID: "plan-the-day", Content: "Plan the day", ProjectID: "project-123", StartDate: start, EndDate: start.AddDate(0, 0, 2), Recurrence: "daily"}
	secondPlan := plan{ID: "plan-weekly-review", Content: "Plan weekly review", ProjectID: "project-123", StartDate: start, EndDate: start, Recurrence: "daily"}
	thirdPlan := plan{ID: "write-report", Content: "Write report", ProjectID: "project-123", StartDate: start, EndDate: start, Recurrence: "daily"}
	for _, schedule := range []plan{p, secondPlan, thirdPlan} {
		if err := addPlan(schedule); err != nil {
			t.Fatal(err)
		}
	}
	all, err := plans()
	if err != nil || len(all) != 3 {
		t.Fatalf("unexpected schedules: %#v, %v", all, err)
	}
	firstPage, err := plansPage("plan", 1, 0)
	if err != nil || len(firstPage) != 1 || firstPage[0].Content != p.Content {
		t.Fatalf("unexpected first filtered page: %#v, %v", firstPage, err)
	}
	secondPage, err := plansPage("plan", 1, 1)
	if err != nil || len(secondPage) != 1 || secondPage[0].Content != secondPlan.Content {
		t.Fatalf("unexpected second filtered page: %#v, %v", secondPage, err)
	}
	if err := recordTodoistTask(p.ID, start, "todoist-1"); err != nil {
		t.Fatal(err)
	}
	ids, err := todoistTaskIDs(p.ID)
	if err != nil || len(ids) != 1 || ids[0] != "todoist-1" {
		t.Fatalf("unexpected Todoist task IDs: %#v, %v", ids, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != "/tasks/todoist-1" {
			t.Fatalf("unexpected Todoist delete request: %s %s", request.Method, request.URL.Path)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	originalURL := todoistAPIBaseURL
	todoistAPIBaseURL = server.URL
	t.Cleanup(func() { todoistAPIBaseURL = originalURL })
	t.Setenv("TODOIST_API_TOKEN", "test-token")
	deleted, err := deletePlanAndTodoistTasks(p)
	if err != nil || deleted != 1 {
		t.Fatalf("unexpected deletion result: %d, %v", deleted, err)
	}
	for _, schedule := range []plan{secondPlan, thirdPlan} {
		if err := removePlan(schedule.ID); err != nil {
			t.Fatal(err)
		}
	}
}
