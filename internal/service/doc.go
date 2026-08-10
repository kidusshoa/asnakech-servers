// Package service contains application use-cases / business workflows.
//
// Services orchestrate domain logic and repositories. They are the only
// layer handlers should call for non-trivial work. Keep handlers thin:
// bind → call service → map to response.
package service
