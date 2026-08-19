package main

import (
	"strings"
	"testing"
)

func TestAccessTokenPrefersEnvironmentToken(t *testing.T) {
	t.Setenv("TODOIST_API_TOKEN", "direct-token")

	token, err := accessToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "direct-token" {
		t.Fatalf("got %q, want direct-token", token)
	}
}

func TestRandomValueProducesURLSafeNonce(t *testing.T) {
	first, second := randomValue(), randomValue()
	if first == second || len(first) < 40 {
		t.Fatalf("expected distinct nonces, got %q and %q", first, second)
	}
	if strings.ContainsAny(first, "+/=") {
		t.Fatalf("nonce is not URL safe: %q", first)
	}
}
