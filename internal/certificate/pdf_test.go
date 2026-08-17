package certificate_test

import (
	"testing"
	"time"

	"github.com/asnakech/asnakech-servers/internal/certificate"
)

func TestGeneratePDF(t *testing.T) {
	data, err := certificate.GeneratePDF(certificate.PDFData{
		LearnerName:      "Ada Student",
		CourseTitle:      "Intro to Go",
		IssuedAt:         time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC),
		VerificationCode: "ABC123DEF456",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 || data[0] != '%' {
		t.Fatalf("expected PDF header, got len=%d first=%q", len(data), string(data[:min(4, len(data))]))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
