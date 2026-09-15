package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func defaultProjectPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "task-planner", "default-project-id"), nil
}

func readDefaultProjectID() (string, error) {
	path, err := defaultProjectPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(raw))
	if strings.ContainsAny(id, "\r\n") {
		return "", fmt.Errorf("%s must contain one Todoist project ID", path)
	}
	return id, nil
}

func resolveDefaultProject(projects []project, value string) (project, error) {
	value = strings.TrimSpace(value)
	for _, project := range projects {
		if project.ID == value {
			return project, nil
		}
	}
	var match project
	for _, candidate := range projects {
		if strings.EqualFold(candidate.Name, value) {
			if match.ID != "" {
				return project{}, fmt.Errorf("multiple Todoist projects are named %q; use an ID from `task-planner auth projects`", value)
			}
			match = candidate
		}
	}
	if match.ID == "" {
		return project{}, fmt.Errorf("project %q was not found in Todoist; list projects with `task-planner auth projects`", value)
	}
	return match, nil
}

func setDefaultProject(value string) error {
	projects, err := todoistProjects()
	if err != nil {
		return err
	}
	selected, err := resolveDefaultProject(projects, value)
	if err != nil {
		return err
	}
	return saveDefaultProject(selected)
}

func saveDefaultProject(selected project) error {
	path, err := defaultProjectPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(selected.ID+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Chmod(path, 0600); err != nil {
		return err
	}
	fmt.Printf("Default destination project: %s (%s)\n", selected.Name, selected.ID)
	return nil
}

func clearDefaultProject() error {
	path, err := defaultProjectPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Println("Default destination project cleared.")
	return nil
}
