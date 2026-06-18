package composer

import (
	"strings"
	"testing"

	"github.com/dklinux7/devkit/internal/devctx"
)

func TestCompose_Basic(t *testing.T) {
	src := &devctx.Sources{
		Identity: [][]byte{[]byte("Be concise."), []byte("Use Go.")},
		Context:  []byte("Acme Corp."),
		Donts:    []byte("No secrets."),
	}

	r, err := Compose(src, false)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if !strings.HasPrefix(r.Content, Header) {
		t.Fatal("missing header")
	}
	if !strings.Contains(r.Content, "Be concise.\n\nUse Go.\n\nAcme Corp.\n\nNo secrets.") {
		t.Fatalf("unexpected composition:\n%s", r.Content)
	}
}

func TestCompose_WithLessons(t *testing.T) {
	src := &devctx.Sources{
		Identity: [][]byte{[]byte("Identity.")},
		Context:  []byte("Context."),
		Donts:    []byte("Donts."),
		Lessons:  [][]byte{[]byte("Lesson 1."), []byte("Lesson 2.")},
	}

	r, err := Compose(src, false)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if !strings.Contains(r.Content, "Donts.\n\nLesson 1.\n\nLesson 2.") {
		t.Fatalf("lessons should come after donts:\n%s", r.Content)
	}
}

func TestCompose_SizeWarning(t *testing.T) {
	big := strings.Repeat("x", 17*1024)
	src := &devctx.Sources{
		Identity: [][]byte{[]byte(big)},
	}

	r, err := Compose(src, false)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if len(r.Warnings) == 0 {
		t.Fatal("expected size warning")
	}
}

func TestCompose_SizeFail(t *testing.T) {
	big := strings.Repeat("x", 33*1024)
	src := &devctx.Sources{
		Identity: [][]byte{[]byte(big)},
	}

	_, err := Compose(src, false)
	if err == nil {
		t.Fatal("expected error for oversized output")
	}
}

func TestCompose_SizeFailForce(t *testing.T) {
	big := strings.Repeat("x", 33*1024)
	src := &devctx.Sources{
		Identity: [][]byte{[]byte(big)},
	}

	r, err := Compose(src, true)
	if err != nil {
		t.Fatalf("Compose with force: %v", err)
	}
	if r.Size < 33*1024 {
		t.Fatalf("expected large output, got %d bytes", r.Size)
	}
}

func TestCompose_EmptySectionsSkipped(t *testing.T) {
	src := &devctx.Sources{
		Identity: [][]byte{[]byte("Identity.")},
		Context:  nil,
		Donts:    []byte(""),
	}

	r, err := Compose(src, false)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if strings.Contains(r.Content, "\n\n\n") {
		t.Fatal("should not have triple newlines from empty sections")
	}
}
