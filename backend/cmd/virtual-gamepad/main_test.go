package main

import "testing"

func TestJoinTextUsesFullPublicURLWithoutAppendingPort(t *testing.T) {
	got := joinText("https://gooner.casa", 8788, "abc")
	want := "Join: https://gooner.casa?token=abc"
	if got != want {
		t.Fatalf("joinText() = %q, want %q", got, want)
	}
}

func TestJoinTextUsesBarePublicHostnameWithoutAppendingPort(t *testing.T) {
	got := joinText("gooner.casa", 8788, "abc")
	want := "Join: http://gooner.casa?token=abc"
	if got != want {
		t.Fatalf("joinText() = %q, want %q", got, want)
	}
}

func TestJoinTextKeepsLocalhostPort(t *testing.T) {
	got := joinText("localhost", 8788, "abc")
	want := "Join: http://localhost:8788?token=abc"
	if got != want {
		t.Fatalf("joinText() = %q, want %q", got, want)
	}
}
