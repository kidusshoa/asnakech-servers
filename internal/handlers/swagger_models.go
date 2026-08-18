package handlers

import (
	"github.com/asnakech/asnakech-servers/internal/platform/ready"
	"github.com/asnakech/asnakech-servers/internal/response"
)

// Swagger / OpenAPI response models (exported for swag).

// HealthData is the liveness payload.
type HealthData struct {
	Status  string `json:"status" example:"ok"`
	Version string `json:"version" example:"0.1.0"`
}

// HealthResponse wraps liveness in the standard envelope.
type HealthResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    HealthData `json:"data"`
}

// ReadyData is the readiness payload.
type ReadyData struct {
	Status  string         `json:"status" example:"ready"`
	Version string         `json:"version" example:"0.1.0"`
	Checks  []ready.Status `json:"checks"`
}

// ReadyResponse wraps readiness in the standard envelope.
type ReadyResponse struct {
	Success bool      `json:"success" example:"true"`
	Data    ReadyData `json:"data"`
}

// WelcomeData is the API root payload.
type WelcomeData struct {
	Message string `json:"message" example:"Welcome to Asnakech School API"`
	Version string `json:"version" example:"0.1.0"`
}

// WelcomeResponse wraps welcome in the standard envelope.
type WelcomeResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    WelcomeData `json:"data"`
}

// RoleResponse is a single platform role.
type RoleResponse struct {
	ID          string `json:"id" example:"a8b6d3f3-6b9b-42ee-98dc-9d8bf8ba2b0e"`
	Code        string `json:"code" example:"student"`
	Name        string `json:"name" example:"Student"`
	Description string `json:"description" example:"Learner enrolled in courses"`
}

// RolesListResponse wraps a role list in the standard envelope.
type RolesListResponse struct {
	Success bool           `json:"success" example:"true"`
	Data    []RoleResponse `json:"data"`
}

// ErrorResponse is the standard failure envelope.
type ErrorResponse struct {
	Success bool                `json:"success" example:"false"`
	Error   *response.ErrorBody `json:"error"`
	Meta    response.Meta       `json:"meta"`
}
