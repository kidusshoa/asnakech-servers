package live

import (
	"context"
	"fmt"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/domain"
)

// MeetingLink is produced by a live provider adapter.
type MeetingLink struct {
	JoinURL    string
	HostURL    string
	ExternalID string
	Metadata   map[string]string
}

// Provider generates or resolves join links for a session.
type Provider interface {
	Type() domain.LiveProvider
	GenerateLink(ctx context.Context, session *domain.LiveSession) (*MeetingLink, error)
}

// Registry selects the adapter for a session provider.
type Registry struct {
	providers map[domain.LiveProvider]Provider
	defaultP  domain.LiveProvider
}

func NewRegistry(cfg *config.Config) *Registry {
	jitsiBase := strings.TrimRight(cfg.LiveJitsiBaseURL, "/")
	if jitsiBase == "" {
		jitsiBase = "https://meet.jit.si"
	}
	defaultP := domain.LiveProvider(cfg.LiveDefaultProvider)
	if defaultP == "" {
		defaultP = domain.LiveProviderCustom
	}
	return &Registry{
		defaultP: defaultP,
		providers: map[domain.LiveProvider]Provider{
			domain.LiveProviderCustom:     CustomProvider{},
			domain.LiveProviderJitsi:      JitsiProvider{BaseURL: jitsiBase},
			domain.LiveProviderZoom:       ManualProvider{provider: domain.LiveProviderZoom},
			domain.LiveProviderGoogleMeet: ManualProvider{provider: domain.LiveProviderGoogleMeet},
		},
	}
}

func (r *Registry) DefaultProvider() domain.LiveProvider {
	return r.defaultP
}

func (r *Registry) For(p domain.LiveProvider) Provider {
	if p == "" {
		p = r.defaultP
	}
	if prov, ok := r.providers[p]; ok {
		return prov
	}
	return r.providers[domain.LiveProviderCustom]
}

// CustomProvider uses URLs supplied on the session.
type CustomProvider struct{}

func (CustomProvider) Type() domain.LiveProvider { return domain.LiveProviderCustom }

func (CustomProvider) GenerateLink(_ context.Context, session *domain.LiveSession) (*MeetingLink, error) {
	join := strings.TrimSpace(session.JoinURL)
	if join == "" {
		return nil, apperr.Validation("join_url is required for custom provider")
	}
	return &MeetingLink{
		JoinURL: join,
		HostURL: strings.TrimSpace(session.HostURL),
	}, nil
}

// JitsiProvider builds a public Jitsi room URL from the session id.
type JitsiProvider struct {
	BaseURL string
}

func (JitsiProvider) Type() domain.LiveProvider { return domain.LiveProviderJitsi }

func (p JitsiProvider) GenerateLink(_ context.Context, session *domain.LiveSession) (*MeetingLink, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://meet.jit.si"
	}
	room := jitsiRoomName(session.ID)
	join := fmt.Sprintf("%s/%s", base, room)
	return &MeetingLink{
		JoinURL:    join,
		HostURL:    join,
		ExternalID: room,
		Metadata: map[string]string{
			"room": room,
		},
	}, nil
}

// ManualProvider covers Zoom/Meet v1 — paste URLs on the session.
type ManualProvider struct {
	provider domain.LiveProvider
}

func (p ManualProvider) Type() domain.LiveProvider { return p.provider }

func (p ManualProvider) GenerateLink(_ context.Context, session *domain.LiveSession) (*MeetingLink, error) {
	join := strings.TrimSpace(session.JoinURL)
	if join == "" {
		return nil, apperr.Validation(
			fmt.Sprintf("%s integration is manual in v1 — set join_url on the session", p.provider),
		)
	}
	return &MeetingLink{
		JoinURL:    join,
		HostURL:    strings.TrimSpace(session.HostURL),
		ExternalID: strings.TrimSpace(session.ExternalID),
	}, nil
}

func jitsiRoomName(sessionID string) string {
	safe := strings.ReplaceAll(sessionID, "-", "")
	if len(safe) > 20 {
		safe = safe[:20]
	}
	return "asnakech-" + safe
}
