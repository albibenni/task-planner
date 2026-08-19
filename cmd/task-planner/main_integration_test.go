package main

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestSharedPostgresPlansAndClaims(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	os.Setenv("SUPABASE_DB_URL", url)
	if err := databaseReachable(url); err != nil {
		t.Fatalf("database should be reachable: %v", err)
	}
	if err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "truncate task_planner_runs, task_planner_plans")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	p := plan{ID: "plan-the-day", Content: "Plan the day", Period: "day", ProjectID: "project-123"}
	if err := addPlan(p); err != nil {
		t.Fatal(err)
	}
	secondPlan := plan{ID: "plan-weekly-review", Content: "Plan weekly review", Period: "week", ProjectID: "project-123"}
	if err := addPlan(secondPlan); err != nil {
		t.Fatal(err)
	}
	thirdPlan := plan{ID: "write-report", Content: "Write report", Period: "week", ProjectID: "project-123"}
	if err := addPlan(thirdPlan); err != nil {
		t.Fatal(err)
	}
	all, err := plans()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("unexpected plans: %#v", all)
	}
	firstPage, total, err := plansPage("plan", 1, 0)
	if err != nil || total != 2 || len(firstPage) != 1 || firstPage[0].Content != p.Content {
		t.Fatalf("unexpected first filtered page: plans=%#v total=%d err=%v", firstPage, total, err)
	}
	secondPage, total, err := plansPage("plan", 1, 1)
	if err != nil || total != 2 || len(secondPage) != 1 || secondPage[0].Content != secondPlan.Content {
		t.Fatalf("unexpected second filtered page: plans=%#v total=%d err=%v", secondPage, total, err)
	}
	if err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		first, err := claimPlan(ctx, conn, p.Content, "2026-08-19")
		if err != nil || !first {
			t.Fatalf("first claim: %v %t", err, first)
		}
		second, err := claimPlan(ctx, conn, p.Content, "2026-08-19")
		if err != nil || second {
			t.Fatalf("second claim: %v %t", err, second)
		}
		return markCreated(ctx, conn, p.Content, "2026-08-19")
	}); err != nil {
		t.Fatal(err)
	}
	if err := removePlan(p.Content); err != nil {
		t.Fatal(err)
	}
	if err := removePlan(secondPlan.Content); err != nil {
		t.Fatal(err)
	}
	if err := removePlan(thirdPlan.Content); err != nil {
		t.Fatal(err)
	}
}
