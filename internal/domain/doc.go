// Package domain holds core business entities and domain rules for the
// Asnakech education platform (users, courses, enrollments, etc.).
//
// Domain types must not import Gin, SQL drivers, or other infrastructure.
// Handlers, services, and repositories depend on domain — not the reverse.
package domain
