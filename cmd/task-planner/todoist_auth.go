package main

import (
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
	"runtime"
	"sort"
	"strings"
	"time"
)

const secretService = "task-planner.todoist.oauth"

func username() string {
	if currentUser, err := user.Current(); err == nil {
		return currentUser.Username
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

func todoistConfigured() bool {
	if strings.TrimSpace(os.Getenv("TODOIST_API_TOKEN")) != "" {
		return true
	}
	raw, err := readSecret()
	if err != nil {
		return false
	}
	var token tokenStore
	return json.Unmarshal([]byte(raw), &token) == nil && token.AccessToken != ""
}

func randomValue() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func login() error {
	const redirect = "http://localhost:53682/callback"
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
	http.HandleFunc("/callback", func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("state") == state && request.URL.Query().Get("code") != "" {
			fmt.Fprint(w, "<p>Todoist connected. You may close this tab.</p>")
			callback <- request.URL.Query().Get("code")
		} else {
			http.Error(w, "Authorization could not be verified.", http.StatusBadRequest)
		}
		go server.Shutdown(context.Background())
	})
	go server.ListenAndServe()
	authorizeURL, _ := url.Parse("https://app.todoist.com/oauth/authorize")
	query := authorizeURL.Query()
	query.Set("client_id", registration.ClientID)
	query.Set("redirect_uri", redirect)
	query.Set("response_type", "code")
	query.Set("scope", "data:read_write")
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	authorizeURL.RawQuery = query.Encode()
	open := "xdg-open"
	if runtime.GOOS == "darwin" {
		open = "open"
	}
	if err := exec.Command(open, authorizeURL.String()).Start(); err != nil {
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
