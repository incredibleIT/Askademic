package academic

import (
	"context"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return s
}

func TestMigrateCreatesAllTables(t *testing.T) {
	s := newTestStore(t)
	var n int
	if err := s.DB().QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN
		 ('department','major','class','student','teacher','counselor','course',
		  'teaching_class','teaching_class_teacher','enrollment','grade',
		  'program_requirement','graduation_audit','status_change','academic_identity')`,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 15 {
		t.Fatalf("want 15 tables, got %d", n)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}
