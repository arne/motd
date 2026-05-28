package config

import (
	"strings"
	"testing"
)

func TestDefaultBuildsModules(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() returned nil")
	}
	if cfg.Mark == nil {
		t.Fatal("Default() returned a config with no mark — embedded marks are missing")
	}
	if strings.TrimSpace(cfg.Mark.Text) == "" {
		t.Fatal("Default() mark text is empty")
	}
	mods, err := cfg.BuildModules()
	if err != nil {
		t.Fatalf("BuildModules: %v", err)
	}
	for _, name := range []string{"host", "up", "load", "mem", "ip", "disk", "mark"} {
		if _, ok := mods[name]; !ok {
			t.Errorf("expected module %q in default config", name)
		}
	}
}

func TestRandomAnimalMark(t *testing.T) {
	mark, ok := randomAnimalMark()
	if !ok {
		t.Fatal("randomAnimalMark() failed — embed path may be wrong")
	}
	if strings.TrimSpace(mark) == "" {
		t.Fatal("randomAnimalMark() returned blank text")
	}
}

func TestLoadAnimalMark(t *testing.T) {
	mark, ok := loadAnimalMark("mouse")
	if !ok {
		t.Fatal(`loadAnimalMark("mouse") failed — embedded mouse should be present`)
	}
	if strings.TrimSpace(mark) == "" {
		t.Fatal(`loadAnimalMark("mouse") returned blank text`)
	}
	for i, line := range strings.Split(mark, "\n") {
		if !strings.HasPrefix(line, " ") {
			t.Errorf(`loadAnimalMark("mouse") line %d missing leading-space padding: %q`, i, line)
		}
	}

	if _, ok := loadAnimalMark("chicken"); ok {
		t.Error(`loadAnimalMark("chicken") returned true — expected miss for unshipped name`)
	}
	if _, ok := loadAnimalMark("../config"); ok {
		t.Error(`loadAnimalMark("../config") returned true — path traversal should be rejected`)
	}
	if _, ok := loadAnimalMark(""); ok {
		t.Error(`loadAnimalMark("") returned true — empty name should be rejected`)
	}
}
