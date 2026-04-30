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
