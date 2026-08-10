package apperr_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/asnakech/asnakech-servers/internal/apperr"
)

func TestAsExtractsWrappedError(t *testing.T) {
	base := apperr.NotFound("course not found")
	wrapped := errors.Join(errors.New("repo"), base)

	got, ok := apperr.As(wrapped)
	if !ok {
		t.Fatal("expected apperr.Error")
	}
	if got.Code != apperr.CodeNotFound {
		t.Fatalf("code %s", got.Code)
	}
	if got.HTTPStatus != http.StatusNotFound {
		t.Fatalf("status %d", got.HTTPStatus)
	}
}
