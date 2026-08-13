package storage_test

import (
	"testing"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/storage"
)

func TestJoinPublicURL(t *testing.T) {
	got := storage.JoinPublicURL("https://cdn.example.com/asnakech", "http://localhost:9010", "asnakech", "a/b.jpg", true)
	want := "https://cdn.example.com/asnakech/a/b.jpg"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	got = storage.JoinPublicURL("", "http://localhost:9010", "asnakech", "a/b.jpg", true)
	want = "http://localhost:9010/asnakech/a/b.jpg"
	if got != want {
		t.Fatalf("path style got %s want %s", got, want)
	}
}

func TestAllowedContentType(t *testing.T) {
	if !storage.AllowedContentType(domain.MediaPurposeAvatar, "image/png") {
		t.Fatal("avatar png should be allowed")
	}
	if storage.AllowedContentType(domain.MediaPurposeAvatar, "video/mp4") {
		t.Fatal("avatar mp4 should be denied")
	}
	if !storage.AllowedContentType(domain.MediaPurposeLessonMedia, "video/mp4") {
		t.Fatal("lesson video should be allowed")
	}
}
