package main

import "testing"

func TestDatabaseURLUsesEnvironment(t *testing.T) {
	const connectionURL = "postgres://user:password@db.example.com:5432/database"
	t.Setenv("SUPABASE_DB_URL", connectionURL)

	actual, err := dbURL()
	if err != nil {
		t.Fatal(err)
	}
	if actual != connectionURL {
		t.Fatalf("got %q, want %q", actual, connectionURL)
	}
}
