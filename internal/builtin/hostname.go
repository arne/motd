package builtin

import (
	"context"
	"os"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
	"github.com/arne/motd/internal/style"
)

func hostname(_ context.Context, _ module.Spec) (block.Block, error) {
	h, err := os.Hostname()
	if err != nil {
		return block.Block{}, err
	}
	return block.Block{Text: style.Primary.Render(h)}, nil
}
