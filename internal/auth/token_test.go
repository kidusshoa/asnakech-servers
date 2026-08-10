package auth_test

import (
	"testing"
	"time"

	"github.com/asnakech/asnakech-servers/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(hash, "password123") {
		t.Fatal("expected password to match")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("expected password mismatch")
	}
}

func TestOpaqueTokenHashStable(t *testing.T) {
	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" {
		t.Fatal("empty token")
	}
	if auth.HashToken(raw) != hash {
		t.Fatal("hash mismatch")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	tm := auth.NewTokenManager("test-secret", time.Minute, time.Hour)
	token, _, err := tm.IssueAccessToken("user-1", "a@b.com", "student")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.ParseAccessToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.RoleCode != "student" {
		t.Fatalf("claims %+v", claims)
	}
}

func TestAccessTokenRejectsBadSecret(t *testing.T) {
	tm := auth.NewTokenManager("test-secret", time.Minute, time.Hour)
	token, _, err := tm.IssueAccessToken("user-1", "a@b.com", "student")
	if err != nil {
		t.Fatal(err)
	}
	other := auth.NewTokenManager("other-secret", time.Minute, time.Hour)
	if _, err := other.ParseAccessToken(token); err == nil {
		t.Fatal("expected parse failure")
	}
}
