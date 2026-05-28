package config

import (
	"io/fs"
	"math/rand/v2"
	"strings"

	"github.com/arne/motd"
	"github.com/arne/motd/internal/layout"
)

// Default returns a built-in config used when no config file is present.
// It bundles a small set of safe builtins and a randomly-chosen animal mark
// inlined as text, so it works even with no marks on disk.
func Default() *Config {
	cfg := &Config{
		Layout: layout.Node{
			Stack: &layout.Stack{
				Direction: "h",
				Children: []layout.Node{
					{Module: "mark"},
					{Stack: &layout.Stack{
						Direction: "v",
						Children: []layout.Node{
							{Module: "host"},
							{Module: "up"},
							{Module: "load"},
							{Module: "mem"},
							{Module: "ip"},
							{Module: "disk"},
						},
					}},
				},
			},
		},
		Modules: map[string]ModuleSpec{
			"host": {Builtin: "hostname"},
			"up":   {Builtin: "uptime"},
			"load": {Builtin: "load"},
			"mem":  {Builtin: "mem"},
			"ip":   {Builtin: "ip"},
			"disk": {Builtin: "disk"},
		},
	}
	if mark, ok := randomAnimalMark(); ok {
		cfg.Mark = &MarkSpec{Text: mark}
	}
	return cfg
}

// loadAnimalMark returns the embedded mark for the given bare animal name
// (e.g. "mouse"). Returns false if the name is invalid or not shipped, so
// callers can fall back to a random one. The art is left-padded by one
// column so it never butts against the screen edge.
func loadAnimalMark(name string) (string, bool) {
	if name == "" || strings.ContainsAny(name, "/\\.") {
		return "", false
	}
	data, err := motd.Marks.ReadFile("examples/marks/animals/" + name + ".ansi")
	if err != nil {
		return "", false
	}
	art := strings.TrimRight(string(data), "\n")
	lines := strings.Split(art, "\n")
	for i, line := range lines {
		lines[i] = " " + line
	}
	return strings.Join(lines, "\n"), true
}

func randomAnimalMark() (string, bool) {
	entries, err := fs.ReadDir(motd.Marks, "examples/marks/animals")
	if err != nil {
		return "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ansi") {
			names = append(names, strings.TrimSuffix(e.Name(), ".ansi"))
		}
	}
	if len(names) == 0 {
		return "", false
	}
	return loadAnimalMark(names[rand.IntN(len(names))])
}
