package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	c := &Cache{dir: dir}
	n, err := c.Clear()
	if err != nil {
		t.Fatalf("Clear on missing dir: %v", err)
	}
	if n != 0 {
		t.Fatalf("Clear on missing dir: got n=%d, want 0", n)
	}
}

func TestClearRemovesEntriesAndTmpFiles(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("updates.json")
	mustWrite("scrub.json")
	mustWrite("updates.json.tmp") // leftover from an interrupted write
	mustWrite("README.md")        // unrelated file should be left alone

	c := &Cache{dir: dir}
	n, err := c.Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if n != 3 {
		t.Fatalf("Clear: got n=%d, want 3", n)
	}

	remaining, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].Name() != "README.md" {
		names := make([]string, 0, len(remaining))
		for _, e := range remaining {
			names = append(names, e.Name())
		}
		t.Fatalf("after Clear, expected only README.md, got %v", names)
	}
}
