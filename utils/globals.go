package utils

import (
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

var GlobalDebugFlag bool

var (
	StdinIsTerminal  = term.IsTerminal(os.Stdin.Fd())
	StdoutIsTerminal = term.IsTerminal(os.Stdout.Fd())
)

var (
	ColorBlue    = lipgloss.ANSIColor(12)
	ColorGreen   = lipgloss.ANSIColor(10)
	ColorRed     = lipgloss.ANSIColor(9)
	ColorYellow  = lipgloss.ANSIColor(11)
	ColorMagenta = lipgloss.ANSIColor(13)
	ColorCyan    = lipgloss.ANSIColor(14)
	ColorFg      = lipgloss.ANSIColor(15)
	ColorMuted   = lipgloss.ANSIColor(7)
	ColorChrome  = lipgloss.ANSIColor(8)
)
