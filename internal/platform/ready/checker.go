package ready

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Status is the readiness report for a single dependency.
type Status struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Skipped bool   `json:"skipped,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Checker pings configured infrastructure dependencies.
type Checker struct {
	DatabaseURL string
	RedisURL    string
	S3Endpoint  string
	Timeout     time.Duration
}

func (c *Checker) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 2 * time.Second
}

// Check runs all dependency probes and returns per-dependency status plus overall OK.
func (c *Checker) Check(ctx context.Context) (statuses []Status, ok bool) {
	ok = true
	statuses = append(statuses, c.checkPostgres(ctx))
	statuses = append(statuses, c.checkRedis(ctx))
	statuses = append(statuses, c.checkS3(ctx))

	for _, s := range statuses {
		if !s.Skipped && !s.OK {
			ok = false
			break
		}
	}
	return statuses, ok
}

func (c *Checker) checkPostgres(ctx context.Context) Status {
	name := "postgres"
	if c.DatabaseURL == "" {
		return Status{Name: name, Skipped: true, OK: true}
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	pool, err := pgxpool.New(ctx, c.DatabaseURL)
	if err != nil {
		return Status{Name: name, OK: false, Error: err.Error()}
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return Status{Name: name, OK: false, Error: err.Error()}
	}
	return Status{Name: name, OK: true}
}

func (c *Checker) checkRedis(ctx context.Context) Status {
	name := "redis"
	if c.RedisURL == "" {
		return Status{Name: name, Skipped: true, OK: true}
	}

	opts, err := redis.ParseURL(c.RedisURL)
	if err != nil {
		return Status{Name: name, OK: false, Error: err.Error()}
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	client := redis.NewClient(opts)
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		return Status{Name: name, OK: false, Error: err.Error()}
	}
	return Status{Name: name, OK: true}
}

func (c *Checker) checkS3(ctx context.Context) Status {
	name := "s3"
	if c.S3Endpoint == "" {
		return Status{Name: name, Skipped: true, OK: true}
	}

	host, err := hostPortFromURL(c.S3Endpoint)
	if err != nil {
		return Status{Name: name, OK: false, Error: err.Error()}
	}

	d := net.Dialer{Timeout: c.timeout()}
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return Status{Name: name, OK: false, Error: err.Error()}
	}
	_ = conn.Close()
	return Status{Name: name, OK: true}
}

func hostPortFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("missing host in endpoint %q", raw)
	}
	if u.Port() == "" {
		if u.Scheme == "https" {
			host = net.JoinHostPort(u.Hostname(), "443")
		} else {
			host = net.JoinHostPort(u.Hostname(), "80")
		}
	}
	return host, nil
}
