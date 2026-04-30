package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
)

func mem(_ context.Context, _ module.Spec) (block.Block, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return block.Block{}, err
	}
	defer f.Close()

	var totalKB, availKB int64
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			totalKB = v
		case "MemAvailable:":
			availKB = v
		}
	}
	used := totalKB - availKB
	pct := 0
	if totalKB > 0 {
		pct = int(float64(used) * 100 / float64(totalKB))
	}
	return block.Block{Text: fmt.Sprintf("%s / %s   (%d%%)", humanBytesKB(used), humanBytesKB(totalKB), pct)}, nil
}

func humanBytesKB(kb int64) string {
	const (
		MB = 1024
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case kb >= TB:
		return fmt.Sprintf("%.1fT", float64(kb)/float64(TB))
	case kb >= GB:
		return fmt.Sprintf("%.1fG", float64(kb)/float64(GB))
	case kb >= MB:
		return fmt.Sprintf("%.1fM", float64(kb)/float64(MB))
	default:
		return fmt.Sprintf("%dK", kb)
	}
}
