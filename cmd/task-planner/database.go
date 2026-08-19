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
	_, err := conn.Exec(ctx, `create table if not exists task_planner_schedules (id text primary key,content text not null unique,project_id text not null,start_date date not null,end_date date not null check(end_date >= start_date),recurrence text not null check(recurrence in ('daily','alternate','weekdays')),weekdays smallint[] not null default '{}',priority smallint check(priority between 1 and 4),created_at timestamptz not null default now()); create table if not exists task_planner_schedule_tasks (schedule_id text not null references task_planner_schedules(id) on delete cascade,due_date date not null,todoist_task_id text not null unique,primary key(schedule_id,due_date));`)
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
		weekdays := p.Weekdays
		if weekdays == nil {
			weekdays = []int16{}
		}
		tag, err := conn.Exec(ctx, "insert into task_planner_schedules(id,content,project_id,start_date,end_date,recurrence,weekdays,priority) values($1,$2,$3,$4,$5,$6,$7,$8) on conflict(content) do nothing", p.ID, p.Content, p.ProjectID, p.StartDate, p.EndDate, p.Recurrence, weekdays, p.Priority)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("a schedule with that task text already exists")
		}
		return nil
	})
}

func plans() ([]plan, error) {
	return plansPage("", 100_000, 0)
}

func plansPage(query string, limit, offset int) ([]plan, error) {
	var result []plan
	err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		rows, err := conn.Query(ctx, "select id,content,project_id,start_date,end_date,recurrence,weekdays,priority from task_planner_schedules where content ilike $1 order by content limit $2 offset $3", "%"+query+"%", limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p plan
			if err := rows.Scan(&p.ID, &p.Content, &p.ProjectID, &p.StartDate, &p.EndDate, &p.Recurrence, &p.Weekdays, &p.Priority); err != nil {
				return err
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	return result, err
}

func removePlan(id string) error {
	return withDB(func(ctx context.Context, conn *pgx.Conn) error {
		tag, err := conn.Exec(ctx, "delete from task_planner_schedules where id=$1", id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("no active schedule has that task")
		}
		return nil
	})
}

func recordTodoistTask(scheduleID string, dueDate time.Time, todoistTaskID string) error {
	return withDB(func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "insert into task_planner_schedule_tasks(schedule_id,due_date,todoist_task_id) values($1,$2,$3)", scheduleID, dueDate, todoistTaskID)
		return err
	})
}

func todoistTaskIDs(scheduleID string) ([]string, error) {
	var ids []string
	err := withDB(func(ctx context.Context, conn *pgx.Conn) error {
		rows, err := conn.Query(ctx, "select todoist_task_id from task_planner_schedule_tasks where schedule_id=$1 order by due_date", scheduleID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	return ids, err
}
