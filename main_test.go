package main

import "testing"

func TestServerPortUsesConfigPort(t *testing.T) {
	t.Setenv("PORT", "")

	if got := serverPort(":8080"); got != ":8080" {
		t.Fatalf("serverPort() = %q, want %q", got, ":8080")
	}
}

func TestServerPortUsesPortEnv(t *testing.T) {
	t.Setenv("PORT", "8081")

	if got := serverPort(":8080"); got != ":8081" {
		t.Fatalf("serverPort() = %q, want %q", got, ":8081")
	}
}

func TestServerPortKeepsHostAddress(t *testing.T) {
	t.Setenv("PORT", "127.0.0.1:8081")

	if got := serverPort(":8080"); got != "127.0.0.1:8081" {
		t.Fatalf("serverPort() = %q, want %q", got, "127.0.0.1:8081")
	}
}

func TestServerPortFallsBackToDefault(t *testing.T) {
	t.Setenv("PORT", "")

	if got := serverPort(" "); got != ":8080" {
		t.Fatalf("serverPort() = %q, want %q", got, ":8080")
	}
}
