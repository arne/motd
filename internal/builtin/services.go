package builtin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
	"github.com/arne/motd/internal/style"
)

type serviceCheck struct {
	Name string
	HTTP string
	TCP  string
	Unit string
	Cmd  string
}

func servicesFull(ctx context.Context, spec module.Spec, w io.Writer) error {
	checks, err := parseServiceList(spec)
	if err != nil {
		return err
	}
	results := make([]serviceResult, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func(i int, c serviceCheck) {
			defer wg.Done()
			start := time.Now()
			results[i] = runCheck(ctx, c)
			results[i].dur = time.Since(start)
		}(i, c)
	}
	wg.Wait()

	maxName, maxTarget := 0, 0
	for i, r := range results {
		c := checks[i]
		if len(r.name) > maxName {
			maxName = len(r.name)
		}
		t := checkTarget(c)
		if len(t) > maxTarget {
			maxTarget = len(t)
		}
	}
	for i, r := range results {
		c := checks[i]
		glyph := style.CritGlyph
		if r.ok {
			glyph = style.OKGlyph
		}
		target := style.Label.Render(fmt.Sprintf("%-*s", maxTarget, checkTarget(c)))
		dur := style.Hint.Render(r.dur.Round(time.Millisecond).String())
		fmt.Fprintf(w, "%s  %-*s  %s  %s  %s\n",
			glyph, maxName, r.name, target, r.info, dur)
	}
	return nil
}

func checkTarget(c serviceCheck) string {
	switch {
	case c.HTTP != "":
		return c.HTTP
	case c.TCP != "":
		return "tcp:" + c.TCP
	case c.Unit != "":
		return "unit:" + c.Unit
	case c.Cmd != "":
		return "cmd"
	}
	return "?"
}

func services(ctx context.Context, spec module.Spec) (block.Block, error) {
	checks, err := parseServiceList(spec)
	if err != nil {
		return block.Block{}, err
	}

	results := make([]serviceResult, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func(i int, c serviceCheck) {
			defer wg.Done()
			results[i] = runCheck(ctx, c)
		}(i, c)
	}
	wg.Wait()

	var down []string
	for _, r := range results {
		if !r.ok {
			down = append(down, r.name)
		}
	}

	st := &block.Status{Module: spec.Name, HasFull: true}
	if len(down) == 0 {
		st.Severity = block.SevOK
		return block.Block{
			Text:   fmt.Sprintf("%d services running", len(checks)),
			Status: st,
		}, nil
	}

	st.Severity = block.SevWarn
	verb := "service"
	if len(down) > 1 {
		verb = "services"
	}
	return block.Block{
		Text:   fmt.Sprintf("%d %s down: %s", len(down), verb, strings.Join(down, ", ")),
		Status: st,
	}, nil
}

type serviceResult struct {
	name string
	ok   bool
	info string
	dur  time.Duration
}

func runCheck(ctx context.Context, c serviceCheck) serviceResult {
	cctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	switch {
	case c.HTTP != "":
		req, err := http.NewRequestWithContext(cctx, "GET", c.HTTP, nil)
		if err != nil {
			return serviceResult{name: c.Name, info: err.Error()}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return serviceResult{name: c.Name, info: err.Error()}
		}
		resp.Body.Close()
		return serviceResult{
			name: c.Name,
			ok:   resp.StatusCode >= 200 && resp.StatusCode < 400,
			info: fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	case c.TCP != "":
		conn, err := net.DialTimeout("tcp", c.TCP, time.Second)
		if err != nil {
			return serviceResult{name: c.Name, info: err.Error()}
		}
		conn.Close()
		return serviceResult{name: c.Name, ok: true, info: "connected"}
	case c.Unit != "":
		out, err := exec.CommandContext(cctx, "systemctl", "is-active", c.Unit).Output()
		state := strings.TrimSpace(string(out))
		return serviceResult{name: c.Name, ok: err == nil && state == "active", info: state}
	case c.Cmd != "":
		err := exec.CommandContext(cctx, "sh", "-c", c.Cmd).Run()
		return serviceResult{name: c.Name, ok: err == nil}
	}
	return serviceResult{name: c.Name, info: "no check primitive"}
}

func parseServiceList(spec module.Spec) ([]serviceCheck, error) {
	if spec.Raw == nil {
		return nil, nil
	}
	raw, ok := spec.Raw["list"].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]serviceCheck, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := serviceCheck{}
		if v, ok := m["name"].(string); ok {
			c.Name = v
		}
		if v, ok := m["http"].(string); ok {
			c.HTTP = v
		}
		if v, ok := m["tcp"].(string); ok {
			c.TCP = v
		}
		if v, ok := m["unit"].(string); ok {
			c.Unit = v
		}
		if v, ok := m["cmd"].(string); ok {
			c.Cmd = v
		}
		out = append(out, c)
	}
	return out, nil
}
