package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5"
)

const secretService = "task-planner.todoist.oauth"

type tokenStore struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	ClientID     string `json:"client_id"`
	ExpiresAt    int64  `json:"expires_at"`
}

type project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type plan struct {
	ID, Content, Period, ProjectID string
	DueString                      *string
	Priority                       *int16
}

func username() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}
func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

func readSecret() (string, error) {
	if runtime.GOOS == "darwin" {
		return run("security", "find-generic-password", "-a", username(), "-s", secretService, "-w")
	}
	return run("secret-tool", "lookup", "service", secretService, "account", username())
}
func writeSecret(value string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("security", "add-generic-password", "-a", username(), "-s", secretService, "-w", value, "-U").Run()
	}
	cmd := exec.Command("secret-tool", "store", "--label=task-planner Todoist OAuth", "service", secretService, "account", username())
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}
func deleteSecret() error {
	if runtime.GOOS == "darwin" {
		return exec.Command("security", "delete-generic-password", "-a", username(), "-s", secretService).Run()
	}
	return exec.Command("secret-tool", "clear", "service", secretService, "account", username()).Run()
}

func formRequest(values url.Values) (tokenStore, error) {
	resp, err := http.PostForm("https://api.todoist.com/oauth/access_token", values)
	if err != nil {
		return tokenStore{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return tokenStore{}, fmt.Errorf("todoist token request failed: %s", body)
	}
	var token tokenStore
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return tokenStore{}, err
	}
	return token, nil
}
func accessToken() (string, error) {
	if direct := strings.TrimSpace(os.Getenv("TODOIST_API_TOKEN")); direct != "" {
		return direct, nil
	}
	raw, err := readSecret()
	if err != nil {
		return "", errors.New("todoist is not connected; run `task-planner auth login` first")
	}
	var token tokenStore
	if json.Unmarshal([]byte(raw), &token) != nil || token.AccessToken == "" {
		return "", errors.New("todoist credentials are invalid; run `task-planner auth login` first")
	}
	if token.ExpiresAt-time.Now().UnixMilli() > 60_000 {
		return token.AccessToken, nil
	}
	refreshed, err := formRequest(url.Values{"grant_type": {"refresh_token"}, "client_id": {token.ClientID}, "refresh_token": {token.RefreshToken}})
	if err != nil {
		return "", err
	}
	refreshed.ClientID = token.ClientID
	refreshed.ExpiresAt = time.Now().Add(time.Duration(refreshed.ExpiresIn) * time.Second).UnixMilli()
	encoded, _ := json.Marshal(refreshed)
	if err := writeSecret(string(encoded)); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

func randomValue() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func login() error {
	redirect := "http://localhost:53682/callback"
	regBody := strings.NewReader(`{"client_name":"task-planner","redirect_uris":["http://localhost:53682/callback"],"scope":"data:read_write","grant_types":["authorization_code","refresh_token"],"response_types":["code"],"token_endpoint_auth_method":"none"}`)
	resp, err := http.Post("https://api.todoist.com/oauth/register", "application/json", regBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var registration struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registration); err != nil || registration.ClientID == "" {
		return errors.New("todoist client registration failed")
	}
	state, verifier := randomValue(), randomValue()
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	callback := make(chan string, 1)
	server := &http.Server{Addr: "127.0.0.1:53682"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") == state && r.URL.Query().Get("code") != "" {
			fmt.Fprint(w, "<p>Todoist connected. You may close this tab.</p>")
			callback <- r.URL.Query().Get("code")
		} else {
			http.Error(w, "Authorization could not be verified.", http.StatusBadRequest)
		}
		go server.Shutdown(context.Background())
	})
	go server.ListenAndServe()
	u, _ := url.Parse("https://app.todoist.com/oauth/authorize")
	q := u.Query()
	q.Set("client_id", registration.ClientID)
	q.Set("redirect_uri", redirect)
	q.Set("response_type", "code")
	q.Set("scope", "data:read_write")
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	open := "xdg-open"
	if runtime.GOOS == "darwin" {
		open = "open"
	}
	if err := exec.Command(open, u.String()).Start(); err != nil {
		return err
	}
	code := <-callback
	token, err := formRequest(url.Values{"grant_type": {"authorization_code"}, "client_id": {registration.ClientID}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {redirect}})
	if err != nil {
		return err
	}
	token.ClientID = registration.ClientID
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).UnixMilli()
	encoded, _ := json.Marshal(token)
	return writeSecret(string(encoded))
}

func todoistProjects() ([]project, error) {
	token, err := accessToken()
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("GET", "https://api.todoist.com/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Results []project `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	sort.Slice(result.Results, func(i, j int) bool { return result.Results[i].Name < result.Results[j].Name })
	return result.Results, nil
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
	u, err := dbURL()
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, u)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if err = setupDB(ctx, conn); err != nil {
		return err
	}
	return fn(ctx, conn)
}
func slug(text string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(text) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
func addPlan(p plan) error {
	return withDB(func(ctx context.Context, c *pgx.Conn) error {
		tag, err := c.Exec(ctx, "insert into task_planner_plans(content,id,period,project_id,due_string,priority) values($1,$2,$3,$4,$5,$6) on conflict(content) do nothing", p.Content, p.ID, p.Period, p.ProjectID, p.DueString, p.Priority)
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
	err := withDB(func(ctx context.Context, c *pgx.Conn) error {
		rows, err := c.Query(ctx, "select id,content,period,project_id,due_string,priority from task_planner_plans order by content")
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
func removePlan(text string) error {
	return withDB(func(ctx context.Context, c *pgx.Conn) error {
		tag, err := c.Exec(ctx, "delete from task_planner_plans where content=$1", text)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("no active plan has that task text")
		}
		return nil
	})
}

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
func taskMarker(p plan, key string) string { return fmt.Sprintf("[task-planner:%s:%s]", p.ID, key) }
func taskExists(p plan, key string) (bool, error) {
	cursor := ""
	for {
		q := url.Values{"project_id": {p.ProjectID}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var result struct {
			Results []struct {
				Content     string `json:"content"`
				Description string `json:"description"`
			} `json:"results"`
			NextCursor *string `json:"next_cursor"`
		}
		if err := todoistRequest("GET", "/tasks?"+q.Encode(), nil, &result); err != nil {
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
	var result any
	if err := todoistRequest("POST", "/tasks", strings.NewReader(mustJSON(payload)), &result); err != nil {
		return err
	}
	fmt.Println("Created:", p.Content)
	return nil
}
func mustJSON(value any) string { raw, _ := json.Marshal(value); return string(raw) }
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

type addModel struct {
	step, cursor    int
	content, due    string
	period          string
	priority        int16
	projects        []project
	done, cancelled bool
}

func (m addModel) Init() tea.Cmd { return nil }
func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		k := key.String()
		if k == "ctrl+c" || k == "esc" {
			m.cancelled = true
			return m, tea.Quit
		}
		if m.step == 0 {
			if k == "enter" && strings.TrimSpace(m.content) != "" {
				m.step++
				m.cursor = 0
			} else if k == "backspace" && len(m.content) > 0 {
				m.content = m.content[:len(m.content)-1]
			} else if len(k) == 1 {
				m.content += k
			}
			return m, nil
		}
		choices := m.choices()
		if k == "up" || k == "k" {
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
		if k == "down" || k == "j" {
			if m.cursor < len(choices)-1 {
				m.cursor++
			}
			return m, nil
		}
		if k == "enter" {
			switch m.step {
			case 1:
				m.period = choices[m.cursor]
				m.step++
				m.cursor = 0
			case 2:
				m.due = choices[m.cursor]
				m.step++
				m.cursor = 0
			case 3:
				m.priority = int16(m.cursor + 1)
				m.step++
				m.cursor = 0
			case 4:
				m.done = true
				return m, tea.Quit
			}
			return m, nil
		}
	}
	return m, nil
}
func (m addModel) choices() []string {
	switch m.step {
	case 1:
		return []string{"day", "week", "month"}
	case 2:
		return []string{"today", "tomorrow", "No due date"}
	case 3:
		return []string{"1 — normal", "2 — medium", "3 — high", "4 — urgent"}
	case 4:
		out := make([]string, len(m.projects))
		for i, p := range m.projects {
			out[i] = p.Name
		}
		return out
	}
	return nil
}
func (m addModel) View() string {
	if m.done {
		return "Plan ready.\n"
	}
	if m.cancelled {
		return "Cancelled.\n"
	}
	if m.step == 0 {
		return "Add a shared Todoist plan\n\nTask text: " + m.content + "█\n\nEnter to continue · Esc to cancel\n"
	}
	labels := []string{"Period", "Due date", "Priority", "Destination project"}
	var b strings.Builder
	b.WriteString("Add a shared Todoist plan\n\n" + labels[m.step-1] + ":\n\n")
	for i, v := range m.choices() {
		mark := "  "
		if i == m.cursor {
			mark = "› "
		}
		b.WriteString(mark + v + "\n")
	}
	b.WriteString("\n↑/↓ select · Enter continue · Esc cancel\n")
	return b.String()
}
func guidedAdd() error {
	projects, err := todoistProjects()
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(addModel{projects: projects}, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	m := final.(addModel)
	if m.cancelled {
		return nil
	}
	due := m.due
	if due == "No due date" {
		due = ""
	}
	var duePtr *string
	if due != "" {
		duePtr = &due
	}
	project := m.projects[m.cursor]
	id := slug(m.content)
	sum := sha256.Sum256([]byte(m.content))
	id = fmt.Sprintf("%s-%x", id, sum[:4])
	p := plan{ID: id, Content: m.content, Period: m.period, ProjectID: project.ID, DueString: duePtr, Priority: &m.priority}
	if err := addPlan(p); err != nil {
		return err
	}
	fmt.Printf("Added %q to %s.\n", p.Content, project.Name)
	return nil
}

func config() error {
	fmt.Print("Paste the Supabase Session Pooler URL: ")
	value, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return errors.New("enter a valid postgres:// or postgresql:// connection URL")
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "task-planner")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "environment")
	if err = os.WriteFile(path, []byte("SUPABASE_DB_URL="+value+"\n"), 0600); err != nil {
		return err
	}
	_ = os.Chmod(path, 0600)
	os.Setenv("SUPABASE_DB_URL", value)
	if err = withDB(func(context.Context, *pgx.Conn) error { return nil }); err != nil {
		return err
	}
	fmt.Println("Saved connection and initialized shared tables.")
	return nil
}
func usage() {
	fmt.Print(`task-planner — shared Todoist plans

Usage:
  task-planner config          Configure Supabase interactively
  task-planner auth login      Connect Todoist
  task-planner auth projects   List projects
  task-planner add             Add a plan in the guided TUI
  task-planner plans           List active plans
  task-planner delete <text>   Stop a plan by task text
  task-planner run [--dry-run] Run shared plans
`)
}
func main() {
	args := os.Args[1:]
	var err error
	switch {
	case len(args) == 0 || args[0] == "help" || args[0] == "--help":
		usage()
	case len(args) == 1 && args[0] == "config":
		err = config()
	case len(args) == 2 && args[0] == "auth" && args[1] == "login":
		err = login()
	case len(args) == 2 && args[0] == "auth" && args[1] == "logout":
		err = deleteSecret()
	case len(args) == 2 && args[0] == "auth" && args[1] == "projects":
		var p []project
		p, err = todoistProjects()
		if err == nil {
			json.NewEncoder(os.Stdout).Encode(p)
		}
	case len(args) == 1 && args[0] == "add":
		err = guidedAdd()
	case len(args) == 1 && args[0] == "plans":
		var p []plan
		p, err = plans()
		if err == nil {
			for _, v := range p {
				fmt.Printf("%s — %s\n", v.Content, v.Period)
			}
		}
	case len(args) >= 2 && args[0] == "delete":
		err = removePlan(strings.Join(args[1:], " "))
	case len(args) >= 1 && args[0] == "run":
		err = schedule(len(args) == 2 && args[1] == "--dry-run")
	default:
		usage()
		err = errors.New("unknown command")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
