package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var todoistAPIBaseURL = "https://api.todoist.com/api/v1"

func todoistRequest(method, path string, body io.Reader, output any) error {
	token, err := accessToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, todoistAPIBaseURL+path, body)
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
	if output == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func taskMarker(p plan, dueDate time.Time) string {
	return fmt.Sprintf("[task-planner:%s:%s]", p.ID, dueDate.Format(time.DateOnly))
}

func createTodoistTask(p plan, dueDate time.Time) (string, error) {
	payload := map[string]any{
		"content":     p.Content,
		"description": taskMarker(p, dueDate) + "\nCreated by task-planner.",
		"project_id":  p.ProjectID,
		"due_date":    dueDate.Format(time.DateOnly),
		"priority":    p.Priority,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := todoistRequest("POST", "/tasks", strings.NewReader(string(encoded)), &result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("todoist did not return an ID for %q", p.Content)
	}
	return result.ID, nil
}

func createScheduleTasks(p plan) error {
	for _, dueDate := range occurrences(p) {
		todoistID, err := createTodoistTask(p, dueDate)
		if err != nil {
			return err
		}
		if err := recordTodoistTask(p.ID, dueDate, todoistID); err != nil {
			return err
		}
	}
	return nil
}

func deletePlanAndTodoistTasks(p plan) (int, error) {
	ids, err := todoistTaskIDs(p.ID)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := todoistRequest("DELETE", "/tasks/"+id, nil, nil); err != nil {
			return 0, err
		}
	}
	if err := removePlan(p.ID); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func occurrences(p plan) []time.Time {
	var result []time.Time
	weekdaySet := make(map[time.Weekday]bool, len(p.Weekdays))
	for _, day := range p.Weekdays {
		weekdaySet[time.Weekday(day)] = true
	}
	for date, index := p.StartDate, 0; !date.After(p.EndDate); date, index = date.AddDate(0, 0, 1), index+1 {
		switch p.Recurrence {
		case "daily":
			result = append(result, date)
		case "alternate":
			if index%2 == 0 {
				result = append(result, date)
			}
		case "weekdays":
			if weekdaySet[date.Weekday()] {
				result = append(result, date)
			}
		}
	}
	return result
}
