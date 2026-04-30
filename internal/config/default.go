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

func randomAnimalMark() (string, bool) {
	entries, err := fs.ReadDir(motd.Marks, "examples/marks/animals")
	if err != nil {
		return "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ansi") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", false
	}
	pick := names[rand.IntN(len(names))]
	data, err := motd.Marks.ReadFile("examples/marks/animals/" + pick)
	if err != nil {
		return "", false
	}
	return strings.TrimRight(string(data), "\n"), true
}
