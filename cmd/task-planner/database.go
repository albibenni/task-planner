package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func databaseReachable(connectionURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connectionURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	return conn.Ping(ctx)
}

func dbURL() (string, error) {
	if value := strings.TrimSpace(os.Getenv("SUPABASE_DB_URL")); value != "" {
		return value, nil
	}
	home, _ := os.UserHomeDir()
	raw, err := os.ReadFile(filepath.Join(home, ".config", "task-planner", "environment"))
	if err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "SUPABASE_DB_URL=") {
				return strings.TrimPrefix(line, "SUPABASE_DB_URL="), nil
			}
		}
	}
	return "", errors.New("supabase is not configured; run `task-planner config` first")
}

func setupDB(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `create table if not exists task_planner_plans (content text primary key,id text not null unique,period text not null check(period in ('day','week','month')),project_id text not null,due_string text,priority smallint check(priority between 1 and 4),created_at timestamptz not null default now()); create table if not exists task_planner_runs (content text not null references task_planner_plans(content) on delete cascade,period_key text not null,status text not null check(status in ('claimed','created')),claimed_at timestamptz not null default now(),created_at timestamptz,primary key(content,period_key));`)
	return err
}

func withDB(fn func(context.Context, *pgx.Conn) error) error {
	ctx := context.Background()
	connectionURL, err := dbURL()
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, connectionURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if err = setupDB(ctx, conn); err != nil {
		return err
	}
	return fn(ctx, conn)
}

func addPlan(p plan) error {
	return withDB(func(ctx context.Context, conn *pgx.Conn) error {
		tag, err := conn.Exec(ctx, "insert into task_planner_plans(content,id,period,project_id,due_string,priority) values($1,$2,$3,$4,$5,$6) on conflict(content) do nothing", p.Content, p.ID, p.Period, p.ProjectID, p.DueString, p.Priority)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("a plan with that task text already exists")
		}
		return nil
	})
}

func plans() ([]plan, error) {
	var result []plan
	err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		rows, err := conn.Query(ctx, "select id,content,period,project_id,due_string,priority from task_planner_plans order by content")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p plan
			if err := rows.Scan(&p.ID, &p.Content, &p.Period, &p.ProjectID, &p.DueString, &p.Priority); err != nil {
				return err
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	return result, err
}

func plansPage(query string, limit, offset int) ([]plan, int, error) {
	var result []plan
	var total int
	err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		pattern := "%" + query + "%"
		if err := conn.QueryRow(ctx, "select count(*) from task_planner_plans where content ilike $1", pattern).Scan(&total); err != nil {
			return err
		}
		rows, err := conn.Query(ctx, "select id,content,period,project_id,due_string,priority from task_planner_plans where content ilike $1 order by content limit $2 offset $3", pattern, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p plan
			if err := rows.Scan(&p.ID, &p.Content, &p.Period, &p.ProjectID, &p.DueString, &p.Priority); err != nil {
				return err
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	return result, total, err
}

func removePlan(text string) error {
	return withDB(func(ctx context.Context, conn *pgx.Conn) error {
		tag, err := conn.Exec(ctx, "delete from task_planner_plans where content=$1", text)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("no active plan has that task text")
		}
		return nil
	})
}

func claimPlan(ctx context.Context, conn *pgx.Conn, content, key string) (bool, error) {
	tag, err := conn.Exec(ctx, "insert into task_planner_runs(content,period_key,status) values($1,$2,'claimed') on conflict(content,period_key) do nothing", content, key)
	if err != nil || tag.RowsAffected() > 0 {
		return tag.RowsAffected() > 0, err
	}
	tag, err = conn.Exec(ctx, "update task_planner_runs set claimed_at=now() where content=$1 and period_key=$2 and status='claimed' and claimed_at < now() - interval '15 minutes'", content, key)
	return tag.RowsAffected() > 0, err
}

func markCreated(ctx context.Context, conn *pgx.Conn, content, key string) error {
	_, err := conn.Exec(ctx, "update task_planner_runs set status='created',created_at=now() where content=$1 and period_key=$2", content, key)
	return err
}
