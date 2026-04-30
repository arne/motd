package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/arne/motd"
	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/builtin"
	"github.com/arne/motd/internal/cache"
	"github.com/arne/motd/internal/config"
	"github.com/arne/motd/internal/module"
	"github.com/arne/motd/internal/render"
	"github.com/arne/motd/internal/watch"
)

const totalBudget = 3 * time.Second

// Set via -ldflags "-X main.version=..." at build time.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	builtin.Register()

	configFlag := flag.String("config", "", "path to config.yaml")
	watchFlag := flag.Bool("watch", false, "re-render whenever the config or referenced files change")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("motd %s", version)
		if commit != "" {
			fmt.Printf(" (%s)", commit)
		}
		if date != "" {
			fmt.Printf(" built %s", date)
		}
		fmt.Println()
		return
	}

	args := flag.Args()
	if len(args) > 0 && args[0] == "example-config" {
		fmt.Print(motd.ExampleConfig)
		return
	}

	if len(args) > 0 && args[0] == "refresh" {
		if err := refreshCache(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	configPath := *configFlag
	explicit := configPath != ""
	if !explicit {
		configPath = config.DefaultPath()
	}

	if len(args) > 0 {
		if err := drillDown(configPath, explicit, args[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if *watchFlag {
		if err := watchMOTD(configPath, explicit); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := renderMOTD(configPath, explicit, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig loads the config at path. When the path is auto-discovered
// (explicit=false) and the file is missing, it falls back to a built-in
// default so a fresh install still produces something useful.
func loadConfig(path string, explicit bool) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if !explicit && errors.Is(err, os.ErrNotExist) {
		return config.Default(), nil
	}
	return nil, err
}

func renderMOTD(configPath string, explicit bool, out io.Writer) error {
	cfg, err := loadConfig(configPath, explicit)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	mods, err := cfg.BuildModules()
	if err != nil {
		return fmt.Errorf("build modules: %w", err)
	}

	c, err := cache.New(cache.DefaultDir())
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), totalBudget)
	defer cancel()

	blocks := runAllModules(ctx, mods, cfg, c)
	rendered := render.Render(cfg.Layout, blocks)
	fmt.Fprintf(out, "\n%s\n\n", rendered)
	return nil
}

func watchMOTD(configPath string, explicit bool) error {
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no config to watch at %s — create one with: motd example-config > %s", configPath, configPath)
	}
	dirs := []string{filepath.Dir(configPath)}
	if cfg, err := config.Load(configPath); err == nil && cfg.Mark != nil && cfg.Mark.File != "" {
		if d := filepath.Dir(cfg.Mark.File); d != dirs[0] {
			dirs = append(dirs, d)
		}
	}
	loop := &watch.Loop{
		WatchDirs: dirs,
		Render: func(out io.Writer) error {
			return renderMOTD(configPath, explicit, out)
		},
	}
	return loop.Run(context.Background())
}

func runAllModules(ctx context.Context, mods map[string]module.Module, cfg *config.Config, c *cache.Cache) map[string]block.Block {
	results := make(map[string]block.Block, len(mods))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for name, m := range mods {
		wg.Add(1)
		go func(name string, m module.Module) {
			defer wg.Done()
			ttl := time.Duration(0)
			if spec, ok := cfg.ModuleSpec(name); ok {
				ttl, _ = time.ParseDuration(spec.Cache)
			}
			b := c.Resolve(ctx, m, ttl)
			mu.Lock()
			results[name] = b
			mu.Unlock()
		}(name, m)
	}
	wg.Wait()
	return results
}

func drillDown(configPath string, explicit bool, name string) error {
	cfg, err := loadConfig(configPath, explicit)
	if err != nil {
		return err
	}
	spec, ok := cfg.ModuleSpec(name)
	if !ok {
		return fmt.Errorf("unknown module: %s", name)
	}

	if spec.CommandFull != "" {
		return runShellInherit(spec.CommandFull)
	}

	if spec.Builtin != "" {
		if fn, ok := module.BuiltinFull(spec.Builtin); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return fn(ctx, toSpecForBuiltin(name, spec), os.Stdout)
		}
	}

	if spec.Command != "" {
		return runShellInherit(spec.Command)
	}

	return fmt.Errorf("module %q has no drill-down", name)
}

func refreshCache() error {
	dir := cache.DefaultDir()
	c, err := cache.New(dir)
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	n, err := c.Clear()
	if err != nil {
		return fmt.Errorf("clear cache: %w", err)
	}
	noun := "entries"
	if n == 1 {
		noun = "entry"
	}
	fmt.Printf("cleared %d cached %s from %s\n", n, noun, dir)
	return nil
}

func runShellInherit(cmd string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

func toSpecForBuiltin(name string, s config.ModuleSpec) module.Spec {
	return module.Spec{
		Name:    name,
		Builtin: s.Builtin,
		Raw:     s.Raw,
	}
}
