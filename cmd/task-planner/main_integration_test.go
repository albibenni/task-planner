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
	all, err := plans()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Content != p.Content {
		t.Fatalf("unexpected plans: %#v", all)
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
}
