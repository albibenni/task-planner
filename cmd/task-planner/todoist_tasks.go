package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var todoistAPIBaseURL = "https://api.todoist.com/api/v1"

const (
	todoistRequestTimeout = 15 * time.Second
	todoistCreateWorkers  = 4
)

var todoistHTTPClient = &http.Client{Timeout: todoistRequestTimeout}

func todoistRequest(method, path string, body io.Reader, output any) error {
	token, err := accessToken()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), todoistRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, todoistAPIBaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := todoistHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("todoist API returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
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
	if err := todoistRequest("POST", "/tasks", bytes.NewReader(encoded), &result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("todoist did not return an ID for %q", p.Content)
	}
	return result.ID, nil
}

func createScheduleTasks(p plan) error {
	tasks, err := createTodoistTasks(p)
	if err != nil {
		return err
	}
	if err := recordTodoistTasks(p.ID, tasks); err != nil {
		return errors.Join(err, deleteCreatedTodoistTasks(tasks))
	}
	return nil
}

func createTodoistTasks(p plan) ([]todoistTaskRecord, error) {
	dates := occurrences(p)
	tasks := make([]todoistTaskRecord, len(dates))
	jobs := make(chan int)
	errs := make(chan error, 1)
	var workers sync.WaitGroup

	workerCount := todoistCreateWorkers
	if len(dates) < workerCount {
		workerCount = len(dates)
	}
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				id, err := createTodoistTask(p, dates[index])
				if err != nil {
					select {
					case errs <- err:
					default:
					}
					continue
				}
				tasks[index] = todoistTaskRecord{dueDate: dates[index], todoistTaskID: id}
			}
		}()
	}
	for index := range dates {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	select {
	case err := <-errs:
		return nil, errors.Join(err, deleteCreatedTodoistTasks(tasks))
	default:
	}
	return tasks, nil
}

// deleteCreatedTodoistTasks compensates for a failed batch so that a retry
// cannot silently create duplicate remote tasks. It intentionally attempts all
// deletions before returning the first failure.
func deleteCreatedTodoistTasks(tasks []todoistTaskRecord) error {
	var result error
	for _, task := range tasks {
		if task.todoistTaskID == "" {
			continue
		}
		if err := todoistRequest("DELETE", "/tasks/"+task.todoistTaskID, nil, nil); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
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
