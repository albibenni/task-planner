package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
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
	duplicatePlan := plan{ID: "plan-the-day-again", Content: "Plan the day", ProjectID: "project-123", StartDate: start, EndDate: start, Recurrence: "daily"}
	secondPlan := plan{ID: "plan-weekly-review", Content: "Plan weekly review", ProjectID: "project-123", StartDate: start, EndDate: start, Recurrence: "daily"}
	thirdPlan := plan{ID: "write-report", Content: "Write report", ProjectID: "project-123", StartDate: start, EndDate: start, Recurrence: "daily"}
	for _, schedule := range []plan{p, duplicatePlan, secondPlan, thirdPlan} {
		if err := addPlan(schedule); err != nil {
			t.Fatal(err)
		}
	}
	all, err := plans()
	if err != nil || len(all) != 4 {
		t.Fatalf("unexpected schedules: %#v, %v", all, err)
	}
	firstPage, err := plansPage("plan", 1, 0)
	if err != nil || len(firstPage) != 1 || firstPage[0].Content != p.Content {
		t.Fatalf("unexpected first filtered page: %#v, %v", firstPage, err)
	}
	secondPage, err := plansPage("plan", 1, 1)
	if err != nil || len(secondPage) != 1 || secondPage[0].ID != duplicatePlan.ID {
		t.Fatalf("unexpected second filtered page: %#v, %v", secondPage, err)
	}
	matching, err := plansCount("plan")
	if err != nil || matching != 3 {
		t.Fatalf("unexpected database search count: %d, %v", matching, err)
	}
	matching, err = plansCount("%")
	if err != nil || matching != 0 {
		t.Fatalf("search characters should be literal, not SQL wildcards: %d, %v", matching, err)
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
	for _, schedule := range []plan{duplicatePlan, secondPlan, thirdPlan} {
		if err := removePlan(schedule.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDatabaseOnlyDeletionLeavesTodoistUntouched(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	t.Setenv("SUPABASE_DB_URL", url)
	t.Setenv("TODOIST_API_TOKEN", "test-token")
	start := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	id := fmt.Sprintf("database-only-%d", time.Now().UnixNano())
	p := plan{ID: id, Content: "Database-only deletion test " + id, ProjectID: "project-123", StartDate: start, EndDate: start, Recurrence: "daily"}
	if err := addPlan(p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = removePlan(p.ID) })
	if err := recordTodoistTask(p.ID, start, "todoist-task-left-alone"); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	originalURL := todoistAPIBaseURL
	todoistAPIBaseURL = server.URL
	t.Cleanup(func() { todoistAPIBaseURL = originalURL })

	result := deleteDatabaseOnlyPlan(p)().(deleteCompletedMsg)
	if result.err != nil || !result.databaseOnly || result.deletedTasks != 0 {
		t.Fatalf("unexpected database-only deletion result: %#v", result)
	}
	if requests.Load() != 0 {
		t.Fatalf("database-only deletion sent %d Todoist request(s)", requests.Load())
	}
	remaining, err := plansPage(p.Content, 10, 0)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("schedule should be gone from Supabase: %#v, %v", remaining, err)
	}
	ids, err := todoistTaskIDs(p.ID)
	if err != nil || len(ids) != 0 {
		t.Fatalf("stored task IDs should be removed with the schedule: %#v, %v", ids, err)
	}
}

func TestDeleteOldFiltersByEndDateAndNeverTouchesTodoist(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	t.Setenv("SUPABASE_DB_URL", url)
	t.Setenv("TODOIST_API_TOKEN", "test-token")
	today := time.Now().Format(time.DateOnly)
	before, err := pastPlansCount(today)
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprintf("delete-old-%d", time.Now().UnixNano())
	oldDate := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	futureDate := time.Now().AddDate(1, 0, 0)
	old := plan{ID: id + "-past", Content: id + " past", ProjectID: "project-123", StartDate: oldDate, EndDate: oldDate, Recurrence: "daily"}
	future := plan{ID: id + "-future", Content: id + " future", ProjectID: "project-123", StartDate: futureDate, EndDate: futureDate, Recurrence: "daily"}
	for _, schedule := range []plan{old, future} {
		if err := addPlan(schedule); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = removePlan(schedule.ID) })
	}
	if err := recordTodoistTask(old.ID, oldDate, id+"-todoist-task"); err != nil {
		t.Fatal(err)
	}
	count, err := pastPlansCount(today)
	if err != nil || count != before+1 {
		t.Fatalf("only the ended schedule should be counted: %d, %v", count, err)
	}
	page, err := pastPlansPage(today, 100_000, 0)
	if err != nil {
		t.Fatal(err)
	}
	var foundOld, foundFuture bool
	for _, schedule := range page {
		foundOld = foundOld || schedule.ID == old.ID
		foundFuture = foundFuture || schedule.ID == future.ID
	}
	if !foundOld || foundFuture {
		t.Fatalf("delete old should list only ended schedules: old=%t future=%t", foundOld, foundFuture)
	}
	if err := removePastPlan(future.ID, today); err == nil {
		t.Fatal("the database must reject deletion of a schedule whose end date has not passed")
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	originalURL := todoistAPIBaseURL
	todoistAPIBaseURL = server.URL
	t.Cleanup(func() { todoistAPIBaseURL = originalURL })
	result := deleteOldSelectedPlan(old)().(oldDeleteCompletedMsg)
	if result.err != nil || requests.Load() != 0 {
		t.Fatalf("past schedule deletion should use Supabase only: %v, requests=%d", result.err, requests.Load())
	}
	ids, err := todoistTaskIDs(old.ID)
	if err != nil || len(ids) != 0 {
		t.Fatalf("stored task IDs should be removed by the database cascade: %#v, %v", ids, err)
	}
	count, err = pastPlansCount(today)
	if err != nil || count != before {
		t.Fatalf("the old schedule should be gone: %d, %v", count, err)
	}
}
