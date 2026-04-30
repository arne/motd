package style

import "github.com/charmbracelet/lipgloss"

var (
	Label = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	Hint  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	Faint = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	Primary = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	Accent  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	OK   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	Warn = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Crit = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

var (
	OKGlyph   = OK.Render("●")
	WarnGlyph = Warn.Render("⚠")
	CritGlyph = Crit.Render("✗")
)
