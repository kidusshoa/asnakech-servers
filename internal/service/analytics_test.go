package service

import (
	"testing"
	"time"
)

func TestParseReportRange(t *testing.T) {
	filter, err := ParseReportRange("2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatal(err)
	}
	if filter.From == nil || filter.To == nil {
		t.Fatal("expected from and to")
	}
	if filter.From.Format("2006-01-02") != "2026-01-01" {
		t.Fatalf("from %s", filter.From)
	}

	_, err = ParseReportRange("bad", "")
	if err == nil {
		t.Fatal("expected validation error")
	}

	filter, err = ParseReportRange("", "")
	if err != nil {
		t.Fatal(err)
	}
	if filter.From != nil || filter.To != nil {
		t.Fatal("expected empty filter")
	}
	_ = time.Now()
}
