package service_test

import (
	"testing"

	"github.com/asnakech/asnakech-servers/internal/service"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Asnakech Academy": "asnakech-academy",
		"  Hello!! World ": "hello-world",
		"A":                "a",
	}
	for in, want := range cases {
		if got := service.Slugify(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}
