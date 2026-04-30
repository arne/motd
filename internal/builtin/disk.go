package builtin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
	"github.com/arne/motd/internal/style"
)

func disk(_ context.Context, spec module.Spec) (block.Block, error) {
	paths := configuredPaths(spec)
	if len(paths) == 0 {
		paths = autoDetectPaths()
	}

	rows := make([]diskRow, 0, len(paths))
	for _, p := range paths {
		r, err := diskRowFor(p)
		if err != nil {
			continue
		}
		rows = append(rows, r)
	}

	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s / %s  (%d%%)", humanBytes(r.usedBytes), humanBytes(r.totalBytes), r.pct)
	}
	return block.Block{Text: b.String()}, nil
}

func diskFull(ctx context.Context, spec module.Spec, w io.Writer) error {
	paths := configuredPaths(spec)
	if len(paths) == 0 {
		paths = autoDetectPaths()
	}
	mounts := readMounts()

	type row struct {
		path, fstype string
		used, total  int64
		pct          int
	}
	rows := make([]row, 0, len(paths))
	for _, p := range paths {
		var st syscall.Statfs_t
		if err := syscall.Statfs(p, &st); err != nil {
			continue
		}
		total := int64(st.Blocks) * int64(st.Bsize)
		free := int64(st.Bavail) * int64(st.Bsize)
		used := total - free
		pct := 0
		if total > 0 {
			pct = int(float64(used) * 100 / float64(total))
		}
		rows = append(rows, row{path: p, fstype: mounts[p], used: used, total: total, pct: pct})
	}

	maxPath, maxFs := 0, 0
	for _, r := range rows {
		if len(r.path) > maxPath {
			maxPath = len(r.path)
		}
		if len(r.fstype) > maxFs {
			maxFs = len(r.fstype)
		}
	}

	for _, r := range rows {
		sizes := fmt.Sprintf("%s / %s", humanBytes(r.used), humanBytes(r.total))
		bar := renderBar(r.pct, 24)
		fmt.Fprintf(w, "%-*s  %-*s  %14s  %3d%%  %s\n",
			maxPath, r.path, maxFs, r.fstype, sizes, r.pct, bar)
	}

	for _, r := range rows {
		if r.fstype == "zfs" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, style.Label.Render("ZFS datasets:"))
			out, err := exec.CommandContext(ctx, "zfs", "list", "-H", "-o", "name,used,avail,refer").Output()
			if err == nil {
				for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
					fields := strings.Split(line, "\t")
					if len(fields) >= 4 {
						fmt.Fprintf(w, "  %-26s  used %-7s  avail %-7s\n", fields[0], fields[1], fields[2])
					}
				}
			}
			break
		}
	}
	return nil
}

func renderBar(pct, width int) string {
	filled := width * pct / 100
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	st := style.OK
	switch {
	case pct >= 90:
		st = style.Crit
	case pct >= 70:
		st = style.Warn
	}
	return st.Render(strings.Repeat("█", filled)) + style.Hint.Render(strings.Repeat("░", width-filled))
}

func readMounts() map[string]string {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil
	}
	defer f.Close()
	m := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) >= 3 {
			m[fields[1]] = fields[2]
		}
	}
	return m
}

type diskRow struct {
	usedBytes, totalBytes int64
	pct                   int
}

func configuredPaths(spec module.Spec) []string {
	if spec.Raw == nil {
		return nil
	}
	switch v := spec.Raw["paths"].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func autoDetectPaths() []string {
	paths := []string{"/"}
	if out, err := exec.Command("zpool", "list", "-H", "-o", "name").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			paths = append(paths, "/"+line)
		}
	}
	sort.Strings(paths)
	return paths
}

func diskRowFor(path string) (diskRow, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return diskRow{}, err
	}
	total := int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bavail) * int64(st.Bsize)
	used := total - free
	pct := 0
	if total > 0 {
		pct = int(float64(used) * 100 / float64(total))
	}
	return diskRow{usedBytes: used, totalBytes: total, pct: pct}, nil
}

func humanBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = int64(1024) * GB
	)
	switch {
	case b >= TB:
		return formatNum(float64(b)/float64(TB)) + "T"
	case b >= GB:
		return formatNum(float64(b)/float64(GB)) + "G"
	case b >= MB:
		return formatNum(float64(b)/float64(MB)) + "M"
	case b >= KB:
		return formatNum(float64(b)/float64(KB)) + "K"
	default:
		return strconv.FormatInt(b, 10) + "B"
	}
}

func formatNum(n float64) string {
	if n >= 100 {
		return strconv.FormatInt(int64(n+0.5), 10)
	}
	return strconv.FormatFloat(n, 'f', 1, 64)
}
