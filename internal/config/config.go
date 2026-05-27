package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/arne/motd/internal/layout"
	"github.com/arne/motd/internal/module"
)

type Config struct {
	Mark    *MarkSpec               `yaml:"mark,omitempty"`
	Layout  layout.Node             `yaml:"layout"`
	Modules map[string]ModuleSpec   `yaml:"modules"`

	configDir string
}

type MarkSpec struct {
	File string `yaml:"file,omitempty"`
	Text string `yaml:"text,omitempty"`
}

type ModuleSpec struct {
	Builtin     string         `yaml:"builtin,omitempty"`
	Command     string         `yaml:"command,omitempty"`
	CommandFull string         `yaml:"command_full,omitempty"`
	File        string         `yaml:"file,omitempty"`
	Text        string         `yaml:"text,omitempty"`
	Cache       string         `yaml:"cache,omitempty"`
	Timeout     string         `yaml:"timeout,omitempty"`
	Always      bool           `yaml:"always,omitempty"`
	OKText      string         `yaml:"ok_text,omitempty"`
	Raw         map[string]any `yaml:",inline"`
}

func DefaultPath() string {
	if v := os.Getenv("MOTD_CONFIG"); v != "" {
		return v
	}
	if user := userConfigPath(); user != "" {
		if _, err := os.Stat(user); err == nil {
			return user
		}
	}
	if _, err := os.Stat("/etc/xdg/motd/config.yaml"); err == nil {
		return "/etc/xdg/motd/config.yaml"
	}
	return userConfigPath()
}

func userConfigPath() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "motd", "config.yaml")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "motd", "config.yaml")
	}
	return ""
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	c.configDir = filepath.Dir(path)
	if c.Mark != nil && c.Mark.File != "" {
		c.Mark.File = c.resolvePath(c.Mark.File)
	}
	for name, spec := range c.Modules {
		if spec.File != "" {
			spec.File = c.resolvePath(spec.File)
			c.Modules[name] = spec
		}
	}
	return &c, nil
}

func (c *Config) resolvePath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:]
		}
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.configDir, p)
}

// BuildModules constructs Module values from the config.
func (c *Config) BuildModules() (map[string]module.Module, error) {
	out := make(map[string]module.Module, len(c.Modules))
	for name, spec := range c.Modules {
		m, err := module.New(toSpec(name, spec))
		if err != nil {
			return nil, fmt.Errorf("module %q: %w", name, err)
		}
		out[name] = m
	}
	if c.Mark != nil {
		spec := module.Spec{Name: "mark"}
		switch {
		case c.Mark.File != "":
			if _, err := os.Stat(c.Mark.File); err == nil {
				spec.File = c.Mark.File
			} else if mark, ok := randomAnimalMark(); ok {
				spec.Text = mark
			} else {
				spec.File = c.Mark.File
			}
		case c.Mark.Text != "":
			spec.Text = c.Mark.Text
		}
		m, err := module.New(spec)
		if err != nil {
			return nil, fmt.Errorf("mark: %w", err)
		}
		out["mark"] = m
	}
	return out, nil
}

func (c *Config) ModuleSpec(name string) (ModuleSpec, bool) {
	s, ok := c.Modules[name]
	return s, ok
}

func toSpec(name string, s ModuleSpec) module.Spec {
	cache, _ := time.ParseDuration(s.Cache)
	timeout, _ := time.ParseDuration(s.Timeout)
	return module.Spec{
		Name:        name,
		Builtin:     s.Builtin,
		Command:     s.Command,
		CommandFull: s.CommandFull,
		File:        s.File,
		Text:        s.Text,
		Cache:       cache,
		Timeout:     timeout,
		Always:      s.Always,
		OKText:      s.OKText,
		Raw:         s.Raw,
	}
}
