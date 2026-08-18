package metrics

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Default is the process-wide metrics registry.
var Default = New()

var durationBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// Registry collects Prometheus-compatible counters and histograms in memory.
type Registry struct {
	mu sync.Mutex

	start   time.Time
	version string

	requests map[reqKey]uint64
	duration map[reqKey]*histogram
}

type reqKey struct {
	method string
	path   string
	status int
}

type histogram struct {
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
}

// New returns an empty metrics registry.
func New() *Registry {
	return &Registry{
		start:    time.Now(),
		requests: make(map[reqKey]uint64),
		duration: make(map[reqKey]*histogram),
	}
}

// SetVersion records build/version info exposed as asnakech_build_info.
func (r *Registry) SetVersion(version string) {
	r.mu.Lock()
	r.version = version
	r.mu.Unlock()
}

// ObserveRequest records one HTTP request count and duration.
func (r *Registry) ObserveRequest(method, path string, status int, seconds float64) {
	if method == "" {
		method = "UNKNOWN"
	}
	if path == "" {
		path = "unknown"
	}

	key := reqKey{method: method, path: path, status: status}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests[key]++

	h := r.duration[key]
	if h == nil {
		h = &histogram{
			buckets: append([]float64(nil), durationBuckets...),
			counts:  make([]uint64, len(durationBuckets)+1),
		}
		r.duration[key] = h
	}
	h.observe(seconds)
}

func (h *histogram) observe(v float64) {
	h.sum += v
	h.count++
	for i, upper := range h.buckets {
		if v <= upper {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++
}

// WritePrometheus renders metrics in Prometheus text exposition format 0.0.4.
func (r *Registry) WritePrometheus(w io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := fmt.Fprintf(w, "# HELP asnakech_process_start_time_seconds Start time of the process since unix epoch in seconds.\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# TYPE asnakech_process_start_time_seconds gauge\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "asnakech_process_start_time_seconds %.3f\n", float64(r.start.UnixNano())/1e9); err != nil {
		return err
	}

	if r.version != "" {
		if _, err := fmt.Fprintf(w, "# HELP asnakech_build_info Build metadata.\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "# TYPE asnakech_build_info gauge\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "asnakech_build_info{version=%q} 1\n", r.version); err != nil {
			return err
		}
	}

	if err := r.writeCounter(w, "asnakech_http_requests_total", "Total HTTP requests.", r.requests); err != nil {
		return err
	}
	return r.writeHistogram(w, "asnakech_http_request_duration_seconds", "HTTP request latency in seconds.", r.duration)
}

func (r *Registry) writeCounter(w io.Writer, name, help string, data map[reqKey]uint64) error {
	if _, err := fmt.Fprintf(w, "# HELP %s %s\n", name, help); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# TYPE %s counter\n", name); err != nil {
		return err
	}

	keys := make([]reqKey, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		if keys[i].path != keys[j].path {
			return keys[i].path < keys[j].path
		}
		return keys[i].status < keys[j].status
	})

	for _, k := range keys {
		labels := formatLabels(k.method, k.path, k.status)
		if _, err := fmt.Fprintf(w, "%s{%s} %d\n", name, labels, data[k]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) writeHistogram(w io.Writer, name, help string, data map[reqKey]*histogram) error {
	if _, err := fmt.Fprintf(w, "# HELP %s %s\n", name, help); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# TYPE %s histogram\n", name); err != nil {
		return err
	}

	keys := make([]reqKey, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		if keys[i].path != keys[j].path {
			return keys[i].path < keys[j].path
		}
		return keys[i].status < keys[j].status
	})

	for _, k := range keys {
		h := data[k]
		if h == nil {
			continue
		}
		base := formatLabels(k.method, k.path, k.status)
		var cumulative uint64
		for i, upper := range h.buckets {
			cumulative += h.counts[i]
			if _, err := fmt.Fprintf(w, "%s_bucket{%s,le=%q} %d\n", name, base, formatFloat(upper), cumulative); err != nil {
				return err
			}
		}
		cumulative += h.counts[len(h.buckets)]
		if _, err := fmt.Fprintf(w, "%s_bucket{%s,le=%q} %d\n", name, base, "+Inf", cumulative); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s_sum{%s} %g\n", name, base, h.sum); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s_count{%s} %d\n", name, base, h.count); err != nil {
			return err
		}
	}
	return nil
}

func formatLabels(method, path string, status int) string {
	return fmt.Sprintf(`method=%q,path=%q,status=%q`, method, path, strconv.Itoa(status))
}

func formatFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", v), "0"), ".")
}
