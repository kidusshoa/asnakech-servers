package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/ping", func(c *gin.Context) {
		rid := c.GetString("request_id")
		if rid == "" {
			t.Fatal("expected request_id in context")
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestRequestIDReusesClientHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "client-rid-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "client-rid-123" {
		t.Fatalf("got %q, want client-rid-123", got)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"https://app.asnakech.com"},
		AllowCredentials: true,
	}))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://app.asnakech.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.asnakech.com" {
		t.Fatalf("got Allow-Origin %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("got Allow-Credentials %q", got)
	}
}

func TestSecurityHeadersApplied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.SecurityHeaders(middleware.SecurityHeadersConfig{HSTS: true}))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	for _, hdr := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Content-Security-Policy",
		"Strict-Transport-Security",
	} {
		if got := w.Header().Get(hdr); got == "" {
			t.Fatalf("expected %s header", hdr)
		}
	}
}

func TestSkipPaths(t *testing.T) {
	skips := []string{"/health", "/metrics", "/swagger"}
	cases := map[string]bool{
		"/health":             true,
		"/metrics":            true,
		"/swagger/index.html": true,
		"/api/v1/courses":     false,
	}
	for path, want := range cases {
		if got := middleware.SkipPaths(path, skips); got != want {
			t.Fatalf("SkipPaths(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestCORSStarDisablesCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true, // should be ignored when origin is *
	}))
	r.OPTIONS("/ping", func(c *gin.Context) {})
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("got Allow-Origin %q, want *", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("credentials must not be set with *, got %q", got)
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status %d", w.Code)
	}
}
