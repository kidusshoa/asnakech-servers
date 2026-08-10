package domain

import "time"

// RoleCode is a stable machine-readable role identifier.
type RoleCode string

const (
	RoleStudent RoleCode = "student"
	RoleTeacher RoleCode = "teacher"
	RoleAdmin   RoleCode = "admin"
	RoleParent  RoleCode = "parent"
)

// Role is a platform access role assigned to users.
type Role struct {
	ID          string
	Code        RoleCode
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
