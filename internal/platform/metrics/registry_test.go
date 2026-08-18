package metrics_test

import (
	"strings"
	"testing"

	"github.com/asnakech/asnakech-servers/internal/platform/metrics"
)

func TestRegistryWritePrometheus(t *testing.T) {
	reg := metrics.New()
	reg.SetVersion("0.2.0-test")
	reg.ObserveRequest("GET", "/api/v1/courses/:id", 200, 0.042)
	reg.ObserveRequest("GET", "/api/v1/courses/:id", 404, 0.010)

	var b strings.Builder
	if err := reg.WritePrometheus(&b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"asnakech_process_start_time_seconds",
		"asnakech_build_info{version=\"0.2.0-test\"} 1",
		"asnakech_http_requests_total",
		`method="GET",path="/api/v1/courses/:id",status="200"`,
		"asnakech_http_request_duration_seconds_bucket",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}
