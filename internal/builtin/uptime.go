package builtin

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
)

func uptime(_ context.Context, _ module.Spec) (block.Block, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return block.Block{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return block.Block{}, fmt.Errorf("malformed /proc/uptime")
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return block.Block{}, err
	}
	return block.Block{Text: formatDuration(time.Duration(secs * float64(time.Second)))}, nil
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", minutes))
	return strings.Join(parts, " ")
}
