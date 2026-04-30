package builtin

import (
	"context"
	"os"
	"strings"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
)

func load(_ context.Context, _ module.Spec) (block.Block, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return block.Block{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return block.Block{Text: strings.TrimSpace(string(data))}, nil
	}
	return block.Block{Text: strings.Join(fields[:3], "  ")}, nil
}
