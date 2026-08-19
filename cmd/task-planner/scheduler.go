package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func periodKey(period string, now time.Time) string {
	year, week := now.ISOWeek()
	switch period {
	case "day":
		return now.Format("2006-01-02")
	case "month":
		return now.Format("2006-01")
	default:
		return fmt.Sprintf("%04d-W%02d", year, week)
	}
}

func todoistRequest(method, path string, body io.Reader, output any) error {
	token, err := accessToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, "https://api.todoist.com/api/v1"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("todoist API returned %s: %s", resp.Status, raw)
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func taskMarker(p plan, key string) string {
	return fmt.Sprintf("[task-planner:%s:%s]", p.ID, key)
}

func taskExists(p plan, key string) (bool, error) {
	cursor := ""
	for {
		query := url.Values{"project_id": {p.ProjectID}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		var result struct {
			Results []struct {
				Content     string `json:"content"`
				Description string `json:"description"`
			} `json:"results"`
			NextCursor *string `json:"next_cursor"`
		}
		if err := todoistRequest("GET", "/tasks?"+query.Encode(), nil, &result); err != nil {
			return false, err
		}
		marker := taskMarker(p, key)
		for _, task := range result.Results {
			if strings.Contains(task.Content, marker) || strings.Contains(task.Description, marker) {
				return true, nil
			}
		}
		if result.NextCursor == nil || *result.NextCursor == "" {
			return false, nil
		}
		cursor = *result.NextCursor
	}
}

func createTask(p plan, key string, dry bool) error {
	description := taskMarker(p, key) + "\nCreated automatically by task-planner."
	if dry {
		fmt.Printf("[dry-run] Would create: %s\n", p.Content)
		return nil
	}
	payload := map[string]any{"content": p.Content, "description": description, "project_id": p.ProjectID, "due_string": p.DueString, "priority": p.Priority}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var result any
	if err := todoistRequest("POST", "/tasks", strings.NewReader(string(encoded)), &result); err != nil {
		return err
	}
	fmt.Println("Created:", p.Content)
	return nil
}

func schedule(dry bool) error {
	return withDB(func(ctx context.Context, conn *pgx.Conn) error {
		all, err := plans()
		if err != nil {
			return err
		}
		for _, p := range all {
			key := periodKey(p.Period, time.Now())
			if dry {
				if err := createTask(p, key, true); err != nil {
					return err
				}
				continue
			}
			claimed, err := claimPlan(ctx, conn, p.Content, key)
			if err != nil {
				return err
			}
			if !claimed {
				fmt.Println("Already claimed:", p.Content)
				continue
			}
			exists, err := taskExists(p, key)
			if err != nil {
				return err
			}
			if exists {
				fmt.Println("Recorded existing task:", p.Content)
			} else if err := createTask(p, key, false); err != nil {
				return err
			}
			if err := markCreated(ctx, conn, p.Content, key); err != nil {
				return err
			}
		}
		return nil
	})
}
