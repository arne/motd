package block

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Block struct {
	Text   string
	Status *Status
}

type Severity int

const (
	SevNone Severity = iota
	SevOK
	SevWarn
	SevCrit
)

type Status struct {
	Severity Severity
	Module   string
	HasFull  bool
}

func (b Block) Empty() bool {
	return b.Text == "" && b.Status == nil
}

func (b Block) Width() int {
	w := 0
	for _, line := range strings.Split(b.Text, "\n") {
		if n := lipgloss.Width(line); n > w {
			w = n
		}
	}
	return w
}

func (b Block) Height() int {
	if b.Text == "" {
		return 0
	}
	return strings.Count(b.Text, "\n") + 1
}
