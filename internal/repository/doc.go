// Package repository defines persistence interfaces and their implementations.
//
// Conventions:
//   - Interfaces live here (or next to the domain they persist).
//   - Postgres (and later other) implementations sit in subpackages
//     (e.g. repository/postgres) once a database is introduced.
//   - Repositories accept and return domain types, never HTTP types.
package repository
