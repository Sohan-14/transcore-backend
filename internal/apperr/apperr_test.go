package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewAndAs(t *testing.T) {
	err := New(KindConflict, "email_taken", "already exists")

	ae, ok := As(err)
	if !ok {
		t.Fatal("As must find the error")
	}
	if ae.Kind != KindConflict || ae.Code != "email_taken" {
		t.Fatalf("got %+v", ae)
	}
}

func TestAsThroughWrapping(t *testing.T) {
	base := New(KindNotFound, "missing", "gone")
	wrapped := fmt.Errorf("loading user: %w", base)

	ae, ok := As(wrapped)
	if !ok {
		t.Fatal("As must unwrap")
	}
	if ae.Kind != KindNotFound {
		t.Fatalf("kind = %s", ae.Kind)
	}
}

func TestAsOnPlainError(t *testing.T) {
	if _, ok := As(errors.New("plain")); ok {
		t.Fatal("a plain error must not match")
	}
}

func TestWithFieldsCopies(t *testing.T) {
	base := New(KindInvalid, "bad", "invalid")
	with := base.WithFields(Field{Name: "email", Issue: "required"})

	if len(base.Fields) != 0 {
		t.Fatal("the original must be unchanged")
	}
	if len(with.Fields) != 1 || with.Fields[0].Name != "email" {
		t.Fatalf("got %+v", with.Fields)
	}
}

func TestInternalHidesCause(t *testing.T) {
	cause := errors.New("secret db detail")
	err := Internal(cause)

	if err.Kind != KindInternal {
		t.Fatalf("kind = %s", err.Kind)
	}
	if err.Message != "internal server error" {
		t.Fatalf("message must stay generic, got %q", err.Message)
	}
	if !errors.Is(err.Unwrap(), cause) {
		t.Fatal("cause must be kept for logging")
	}
}
