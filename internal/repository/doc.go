// Package repository defines persistence interfaces and their implementations.
//
// Conventions:
//   - Interfaces live in this package (e.g. RoleRepository).
//   - Postgres implementations live in repository/postgres.
//   - Repositories accept and return domain types, never HTTP types.
package repository
