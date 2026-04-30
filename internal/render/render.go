package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/layout"
	"github.com/arne/motd/internal/style"
)

var (
	ansiRe  = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	parenRe = regexp.MustCompile(`\(([^()]*)\)`)
	unitRe  = regexp.MustCompile(`(\d(?:\.\d+)?)([KMGTdhms])\b`)
	slashRe = regexp.MustCompile(` / `)
)

// dimSecondary styles units, slash separators, and parenthesized content.
// Escape sequences in the input are preserved untouched so styling applied
// upstream (e.g. by builtins) survives.
func dimSecondary(s string) string {
	var b strings.Builder
	last := 0
	for _, m := range ansiRe.FindAllStringIndex(s, -1) {
		b.WriteString(processPlain(s[last:m[0]]))
		b.WriteString(s[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(processPlain(s[last:]))
	return b.String()
}

func processPlain(s string) string {
	s = unitRe.ReplaceAllStringFunc(s, func(m string) string {
		unit := m[len(m)-1:]
		return m[:len(m)-1] + style.Hint.Render(unit)
	})
	s = slashRe.ReplaceAllString(s, " "+style.Hint.Render("/")+" ")
	return parenRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[1 : len(m)-1]
		return style.Faint.Render("(") + style.Accent.Render(inner) + style.Faint.Render(")")
	})
}

func Render(root layout.Node, blocks map[string]block.Block) string {
	return renderNode(root, blocks)
}

func renderNode(n layout.Node, blocks map[string]block.Block) string {
	if n.Module != "" {
		b := blocks[n.Module]
		return b.Text
	}
	if n.Stack == nil {
		return ""
	}
	if n.Stack.Direction == "h" {
		return renderHStack(n.Stack, blocks)
	}
	return renderVStack(n.Stack, blocks)
}

func renderHStack(s *layout.Stack, blocks map[string]block.Block) string {
	parts := make([]string, 0, len(s.Children))
	for _, c := range s.Children {
		r := renderNode(c, blocks)
		if r == "" {
			continue
		}
		parts = append(parts, r)
	}
	if len(parts) == 0 {
		return ""
	}
	gap := "    "
	withGaps := make([]string, 0, len(parts)*2-1)
	for i, p := range parts {
		if i > 0 {
			withGaps = append(withGaps, gap)
		}
		withGaps = append(withGaps, p)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, withGaps...)
}

func renderVStack(s *layout.Stack, blocks map[string]block.Block) string {
	mode := detectVStackMode(s, blocks)
	switch mode {
	case "status":
		return renderStatusVStack(s, blocks)
	case "labeled":
		return renderLabeledVStack(s, blocks)
	default:
		return renderPlainVStack(s, blocks)
	}
}

func detectVStackMode(s *layout.Stack, blocks map[string]block.Block) string {
	hasStatus := false
	hasLeafModule := false
	allEmpty := true
	for _, c := range s.Children {
		if c.Module == "" {
			continue
		}
		hasLeafModule = true
		b := blocks[c.Module]
		if b.Status != nil {
			hasStatus = true
		}
		if b.Text != "" {
			allEmpty = false
		}
	}
	if !hasLeafModule {
		return "plain"
	}
	if hasStatus && !allEmpty {
		return "status"
	}
	if hasLeafModule {
		return "labeled"
	}
	return "plain"
}

func renderStatusVStack(s *layout.Stack, blocks map[string]block.Block) string {
	var lines []string
	for _, c := range s.Children {
		if c.Module == "" {
			lines = append(lines, renderNode(c, blocks))
			continue
		}
		b := blocks[c.Module]
		if b.Empty() {
			continue
		}
		lines = append(lines, formatStatusLine(c.Module, b))
	}
	return strings.Join(lines, "\n")
}

func formatStatusLine(name string, b block.Block) string {
	glyph := style.WarnGlyph
	if b.Status != nil {
		switch b.Status.Severity {
		case block.SevOK:
			glyph = style.OKGlyph
		case block.SevCrit:
			glyph = style.CritGlyph
		}
	}
	hint := ""
	if b.Status != nil && b.Status.HasFull {
		hint = " " + style.Hint.Render(fmt.Sprintf("→ motd %s", name))
	}
	return fmt.Sprintf(" %s %s%s", glyph, b.Text, hint)
}

func renderLabeledVStack(s *layout.Stack, blocks map[string]block.Block) string {
	maxLabel := 0
	for _, c := range s.Children {
		if c.Module == "" {
			continue
		}
		if len(c.Module) > maxLabel {
			maxLabel = len(c.Module)
		}
	}
	var lines []string
	for _, c := range s.Children {
		if c.Module == "" {
			lines = append(lines, renderNode(c, blocks))
			continue
		}
		b := blocks[c.Module]
		if b.Text == "" {
			continue
		}
		valueLines := strings.Split(dimSecondary(b.Text), "\n")
		label := style.Label.Render(padRight(c.Module, maxLabel))
		lines = append(lines, fmt.Sprintf("%s  %s", label, valueLines[0]))
		for _, vl := range valueLines[1:] {
			lines = append(lines, fmt.Sprintf("%s  %s", strings.Repeat(" ", maxLabel), vl))
		}
	}
	return strings.Join(lines, "\n")
}

func renderPlainVStack(s *layout.Stack, blocks map[string]block.Block) string {
	hasComposite := false
	for _, c := range s.Children {
		if c.Stack != nil {
			hasComposite = true
			break
		}
	}
	sep := "\n"
	if hasComposite {
		sep = "\n\n"
	}
	var parts []string
	for _, c := range s.Children {
		r := renderNode(c, blocks)
		if r == "" {
			continue
		}
		parts = append(parts, r)
	}
	return strings.Join(parts, sep)
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
