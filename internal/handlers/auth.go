package handlers

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email" example:"student@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
	FullName string `json:"full_name" example:"Ada Student"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"student@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type UserPublic struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	FullName      string  `json:"full_name"`
	Role          string  `json:"role"`
	EmailVerified bool    `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type TokenPairResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type" example:"Bearer"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AuthSuccessData struct {
	User                 UserPublic        `json:"user"`
	Tokens               TokenPairResponse `json:"tokens"`
	VerificationTokenDev string            `json:"verification_token,omitempty"`
}

type AuthSuccessResponse struct {
	Success bool            `json:"success" example:"true"`
	Data    AuthSuccessData `json:"data"`
}

type MeResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    UserPublic `json:"data"`
}

type MessageData struct {
	Message        string `json:"message"`
	ResetTokenDev  string `json:"reset_token,omitempty"`
}

type MessageResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    MessageData `json:"data"`
}

// Register godoc
// @Summary      Register
// @Description  Create a student account and return access/refresh tokens. In development, may include verification_token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerRequest true "Registration payload"
// @Success      201 {object} AuthSuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	result, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toAuthSuccessData(result))
}

// Login godoc
// @Summary      Login
// @Description  Authenticate with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "Login payload"
// @Success      200 {object} AuthSuccessResponse
// @Failure      401 {object} ErrorResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAuthSuccessData(result))
}

// Refresh godoc
// @Summary      Refresh tokens
// @Description  Rotate refresh token and issue a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body refreshRequest true "Refresh payload"
// @Success      200 {object} AuthSuccessResponse
// @Failure      401 {object} ErrorResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAuthSuccessData(result))
}

// Logout godoc
// @Summary      Logout
// @Description  Revoke a refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body logoutRequest true "Logout payload"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "logged out"})
}

// Me godoc
// @Summary      Current user
// @Description  Return the authenticated user profile
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} MeResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	user, err := h.auth.Me(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserPublic(user))
}

// ForgotPassword godoc
// @Summary      Forgot password
// @Description  Request a password reset. Always returns a generic message. In development may include reset_token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body forgotPasswordRequest true "Forgot password payload"
// @Success      200 {object} MessageResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	result, err := h.auth.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{
		Message:       result.Message,
		ResetTokenDev: result.ResetTokenDev,
	})
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Set a new password using a reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body resetPasswordRequest true "Reset password payload"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "password updated"})
}

// VerifyEmail godoc
// @Summary      Verify email
// @Description  Confirm email ownership with a verification token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body verifyEmailRequest true "Verify email payload"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      429 {object} ErrorResponse
// @Router       /api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	if err := h.auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "email verified"})
}

func toAuthSuccessData(result *service.AuthResult) AuthSuccessData {
	return AuthSuccessData{
		User: toUserPublic(result.User),
		Tokens: TokenPairResponse{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			TokenType:    result.Tokens.TokenType,
			ExpiresAt:    result.Tokens.ExpiresAt,
		},
		VerificationTokenDev: result.VerificationTokenDev,
	}
}

func toUserPublic(user *domain.User) UserPublic {
	return UserPublic{
		ID:            user.ID,
		Email:         user.Email,
		FullName:      user.FullName,
		Role:          string(user.RoleCode),
		EmailVerified: user.EmailVerifiedAt != nil,
		CreatedAt:     user.CreatedAt,
	}
}
