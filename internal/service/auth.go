package service

import (
	"context"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/auth"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

const (
	minPasswordLen           = 8
	emailVerificationTTL     = 48 * time.Hour
	passwordResetTTL         = 1 * time.Hour
)

type AuthService struct {
	users      repository.UserRepository
	roles      repository.RoleRepository
	refresh    repository.RefreshTokenRepository
	resets     repository.PasswordResetTokenRepository
	verifies   repository.EmailVerificationTokenRepository
	tokens     *auth.TokenManager
	exposeDev  bool
}

func NewAuthService(
	users repository.UserRepository,
	roles repository.RoleRepository,
	refresh repository.RefreshTokenRepository,
	resets repository.PasswordResetTokenRepository,
	verifies repository.EmailVerificationTokenRepository,
	tokens *auth.TokenManager,
	exposeDevTokens bool,
) *AuthService {
	return &AuthService{
		users:     users,
		roles:     roles,
		refresh:   refresh,
		resets:    resets,
		verifies:  verifies,
		tokens:    tokens,
		exposeDev: exposeDevTokens,
	}
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
}

type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AuthResult struct {
	User                 *domain.User
	Tokens               AuthTokens
	VerificationTokenDev string // only set in development
}

type ForgotPasswordResult struct {
	Message         string
	ResetTokenDev   string // only set in development when user exists
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, apperr.Validation("a valid email is required")
	}
	if len(in.Password) < minPasswordLen {
		return nil, apperr.Validation("password must be at least 8 characters")
	}

	role, err := s.roles.GetByCode(ctx, domain.RoleStudent)
	if err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, apperr.Internal("could not hash password")
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(in.FullName),
		RoleID:       role.ID,
		RoleCode:     role.Code,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	verifyRaw, err := s.createEmailVerificationToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	result, err := s.issueSession(ctx, user)
	if err != nil {
		return nil, err
	}
	if s.exposeDev {
		result.VerificationTokenDev = verifyRaw
	}
	return result, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == apperr.CodeNotFound {
			return nil, apperr.Unauthorized("invalid email or password")
		}
		return nil, err
	}
	if !user.IsActive {
		return nil, apperr.Forbidden("account is disabled")
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, apperr.Unauthorized("invalid email or password")
	}
	return s.issueSession(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, apperr.Validation("refresh_token is required")
	}

	stored, err := s.refresh.GetByHash(ctx, auth.HashToken(refreshToken))
	if err != nil {
		return nil, err
	}
	if stored.RevokedAt != nil || time.Now().UTC().After(stored.ExpiresAt) {
		return nil, apperr.Unauthorized("invalid refresh token")
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, apperr.Forbidden("account is disabled")
	}

	_ = s.refresh.Revoke(ctx, stored.TokenHash, time.Now().UTC())
	return s.issueSession(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return apperr.Validation("refresh_token is required")
	}
	return s.refresh.Revoke(ctx, auth.HashToken(refreshToken), time.Now().UTC())
}

func (s *AuthService) Me(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, apperr.Forbidden("account is disabled")
	}
	return user, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) (*ForgotPasswordResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	result := &ForgotPasswordResult{
		Message: "if that email is registered, a reset link will be sent",
	}
	if email == "" {
		return result, nil
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == apperr.CodeNotFound {
			return result, nil
		}
		return nil, err
	}

	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, apperr.Internal("could not create reset token")
	}
	token := &domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(passwordResetTTL),
	}
	if err := s.resets.Create(ctx, token); err != nil {
		return nil, err
	}
	if s.exposeDev {
		result.ResetTokenDev = raw
	}
	return result, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if strings.TrimSpace(rawToken) == "" {
		return apperr.Validation("token is required")
	}
	if len(newPassword) < minPasswordLen {
		return apperr.Validation("password must be at least 8 characters")
	}

	stored, err := s.resets.GetByHash(ctx, auth.HashToken(rawToken))
	if err != nil {
		return err
	}
	if stored.UsedAt != nil || time.Now().UTC().After(stored.ExpiresAt) {
		return apperr.Unauthorized("invalid or expired reset token")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return apperr.Internal("could not hash password")
	}
	if err := s.users.UpdatePasswordHash(ctx, stored.UserID, hash); err != nil {
		return err
	}
	if err := s.resets.MarkUsed(ctx, stored.TokenHash, time.Now().UTC()); err != nil {
		return err
	}
	return s.refresh.RevokeAllForUser(ctx, stored.UserID, time.Now().UTC())
}

func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return apperr.Validation("token is required")
	}

	stored, err := s.verifies.GetByHash(ctx, auth.HashToken(rawToken))
	if err != nil {
		return err
	}
	if stored.UsedAt != nil || time.Now().UTC().After(stored.ExpiresAt) {
		return apperr.Unauthorized("invalid or expired verification token")
	}

	now := time.Now().UTC()
	if err := s.users.MarkEmailVerified(ctx, stored.UserID, now); err != nil {
		return err
	}
	return s.verifies.MarkUsed(ctx, stored.TokenHash, now)
}

func (s *AuthService) issueSession(ctx context.Context, user *domain.User) (*AuthResult, error) {
	access, expiresAt, err := s.tokens.IssueAccessToken(user.ID, user.Email, string(user.RoleCode))
	if err != nil {
		return nil, apperr.Internal("could not issue access token")
	}

	rawRefresh, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, apperr.Internal("could not issue refresh token")
	}

	rt := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(s.tokens.RefreshTTL()),
	}
	if err := s.refresh.Create(ctx, rt); err != nil {
		return nil, err
	}

	return &AuthResult{
		User: user,
		Tokens: AuthTokens{
			AccessToken:  access,
			RefreshToken: rawRefresh,
			TokenType:    "Bearer",
			ExpiresAt:    expiresAt,
		},
	}, nil
}

func (s *AuthService) createEmailVerificationToken(ctx context.Context, userID string) (string, error) {
	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return "", apperr.Internal("could not create verification token")
	}
	token := &domain.EmailVerificationToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(emailVerificationTTL),
	}
	if err := s.verifies.Create(ctx, token); err != nil {
		return "", err
	}
	return raw, nil
}
