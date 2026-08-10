package ready_test

import (
	"context"
	"testing"
	"time"

	"github.com/asnakech/asnakech-servers/internal/platform/ready"
)

func TestCheckSkipsUnconfiguredDeps(t *testing.T) {
	c := &ready.Checker{Timeout: 100 * time.Millisecond}
	statuses, ok := c.Check(context.Background())
	if !ok {
		t.Fatalf("expected overall ok when all skipped, got %#v", statuses)
	}
	for _, s := range statuses {
		if !s.Skipped {
			t.Fatalf("expected skipped: %#v", s)
		}
	}
}
