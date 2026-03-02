package errors_test

import (
	"fmt"
	"testing"

	"github.com/awch-D/ForgeX/forgex-core/errors"
)

func TestNewError(t *testing.T) {
	err := errors.New(errors.ErrInvalidInput, "test error")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	expectedMsg := "[FX-1001] test error"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
	}
}

func TestWrapError(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	err := errors.Wrap(errors.ErrNotFound, "not found test", baseErr)

	expectedMsg := "[FX-1002] not found test: base error"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
	}

	if err.Unwrap() != baseErr {
		t.Errorf("expected wrapped error to be %v, got %v", baseErr, err.Unwrap())
	}
}
