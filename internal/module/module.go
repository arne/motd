package module

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/arne/motd/internal/block"
)

type Module interface {
	Name() string
	Render(ctx context.Context) (block.Block, error)
}

type Spec struct {
	Name        string
	Command     string
	CommandFull string
	File        string
	Text        string
	Builtin     string
	Cache       time.Duration
	Timeout     time.Duration
	Always      bool
	OKText      string
	Style       Style

	// Builtin-specific config blob, parsed by each builtin
	Raw map[string]any
}

type Style struct {
	Fg     string
	Bg     string
	Bold   bool
	Italic bool
	Faint  bool
	Label  string
}

type commandModule struct{ Spec }

func (m *commandModule) Name() string { return m.Spec.Name }
func (m *commandModule) Render(ctx context.Context) (block.Block, error) {
	out, err := runShell(ctx, m.Spec.Command, m.Spec.Timeout)
	if err != nil {
		return block.Block{}, err
	}
	out = strings.TrimRight(out, "\n")
	if out == "" {
		if m.Spec.Always {
			text := m.Spec.OKText
			return block.Block{Text: text, Status: &block.Status{
				Severity: block.SevOK, Module: m.Spec.Name, HasFull: m.Spec.CommandFull != "",
			}}, nil
		}
		return block.Block{}, nil
	}
	return block.Block{Text: out, Status: &block.Status{
		Severity: block.SevWarn, Module: m.Spec.Name, HasFull: m.Spec.CommandFull != "",
	}}, nil
}

type fileModule struct{ Spec }

func (m *fileModule) Name() string { return m.Spec.Name }
func (m *fileModule) Render(ctx context.Context) (block.Block, error) {
	data, err := os.ReadFile(expand(m.Spec.File))
	if err != nil {
		return block.Block{}, err
	}
	return block.Block{Text: strings.TrimRight(string(data), "\n")}, nil
}

type textModule struct{ Spec }

func (m *textModule) Name() string                                 { return m.Spec.Name }
func (m *textModule) Render(_ context.Context) (block.Block, error) {
	return block.Block{Text: m.Spec.Text}, nil
}

type BuiltinFunc func(ctx context.Context, spec Spec) (block.Block, error)
type BuiltinFullFunc func(ctx context.Context, spec Spec, w io.Writer) error

var (
	builtins     = map[string]BuiltinFunc{}
	builtinFulls = map[string]BuiltinFullFunc{}
)

func RegisterBuiltin(name string, fn BuiltinFunc) {
	builtins[name] = fn
}

func RegisterBuiltinFull(name string, fn BuiltinFullFunc) {
	builtinFulls[name] = fn
}

func BuiltinFull(name string) (BuiltinFullFunc, bool) {
	fn, ok := builtinFulls[name]
	return fn, ok
}

type builtinModule struct{ Spec }

func (m *builtinModule) Name() string { return m.Spec.Name }
func (m *builtinModule) Render(ctx context.Context) (block.Block, error) {
	fn, ok := builtins[m.Spec.Builtin]
	if !ok {
		return block.Block{}, fmt.Errorf("unknown builtin %q", m.Spec.Builtin)
	}
	return fn(ctx, m.Spec)
}

func New(spec Spec) (Module, error) {
	switch {
	case spec.Builtin != "":
		return &builtinModule{spec}, nil
	case spec.Command != "":
		return &commandModule{spec}, nil
	case spec.File != "":
		return &fileModule{spec}, nil
	case spec.Text != "":
		return &textModule{spec}, nil
	}
	return nil, fmt.Errorf("module %q has no source (command/file/text/builtin)", spec.Name)
}

func runShell(ctx context.Context, command string, timeout time.Duration) (string, error) {
	if timeout == 0 {
		timeout = 500 * time.Millisecond
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:]
		}
	}
	return p
}
