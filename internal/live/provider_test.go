package live_test

import (
	"context"
	"testing"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/live"
)

func TestJitsiProviderGenerateLink(t *testing.T) {
	p := live.JitsiProvider{BaseURL: "https://meet.jit.si"}
	link, err := p.GenerateLink(context.Background(), &domain.LiveSession{ID: "550e8400-e29b-41d4-a716-446655440000"})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://meet.jit.si/asnakech-550e8400e29b41d4a716"
	if link.JoinURL != want {
		t.Fatalf("got %s want %s", link.JoinURL, want)
	}
}

func TestCustomProviderRequiresJoinURL(t *testing.T) {
	_, err := live.CustomProvider{}.GenerateLink(context.Background(), &domain.LiveSession{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
