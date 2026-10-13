package auth

import (
	"testing"
	"time"
)

func TestTokens(t *testing.T) {
	now := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	tokens := NewTokens("secret-secret-secret-secret-secret", 15*time.Minute)

	token, err := tokens.Issue(42, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := tokens.Parse(token, now.Add(14*time.Minute))
	if err != nil || id != 42 {
		t.Fatalf("Parse = %d, %v; want 42", id, err)
	}
	if _, err := tokens.Parse(token, now.Add(16*time.Minute)); err == nil {
		t.Error("expired token must be rejected")
	}
	other := NewTokens("another-secret-another-secret-xx", 15*time.Minute)
	if _, err := other.Parse(token, now); err == nil {
		t.Error("token signed with another secret must be rejected")
	}
	if _, err := tokens.Parse("garbage", now); err == nil {
		t.Error("garbage must be rejected")
	}
}
