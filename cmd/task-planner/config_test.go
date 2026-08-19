package main

import "testing"

func TestValidateDatabaseURL(t *testing.T) {
	valid := []string{
		"postgres://user:password@db.example.com:5432/postgres",
		"postgresql://user:password@db.example.com/database",
	}
	for _, value := range valid {
		if err := validateDatabaseURL(value); err != nil {
			t.Errorf("expected %q to be valid: %v", value, err)
		}
	}

	for _, value := range []string{"", "https://example.com", "postgres://"} {
		if err := validateDatabaseURL(value); err == nil {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}
