package watch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Loop runs render on every relevant filesystem event under watchDirs,
// debounced. Returns on SIGINT/SIGTERM or when ctx is cancelled.
type Loop struct {
	WatchDirs []string
	Render    func(out io.Writer) error
	Debounce  time.Duration
	Out       io.Writer
}

func (l *Loop) Run(ctx context.Context) error {
	if l.Debounce == 0 {
		l.Debounce = 100 * time.Millisecond
	}
	if l.Out == nil {
		l.Out = os.Stdout
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	defer w.Close()

	seen := map[string]bool{}
	for _, d := range l.WatchDirs {
		d = filepath.Clean(d)
		if seen[d] {
			continue
		}
		seen[d] = true
		if err := w.Add(d); err != nil {
			return fmt.Errorf("watch %s: %w", d, err)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	clearAndRender(l.Out, l.Render)

	var pending *time.Timer
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sigs:
			fmt.Fprintln(l.Out)
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if pending != nil {
				pending.Stop()
			}
			pending = time.AfterFunc(l.Debounce, func() {
				clearAndRender(l.Out, l.Render)
			})
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		}
	}
}

func clearAndRender(out io.Writer, render func(io.Writer) error) {
	fmt.Fprint(out, "\033[H\033[2J")
	if err := render(out); err != nil {
		fmt.Fprintf(out, "render error: %v\n", err)
	}
}
