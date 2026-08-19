package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

func config() error {
	fmt.Print("Paste the Supabase Session Pooler URL: ")
	value, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	connectionURL, err := url.Parse(value)
	if err != nil || connectionURL.Host == "" || (connectionURL.Scheme != "postgres" && connectionURL.Scheme != "postgresql") {
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
