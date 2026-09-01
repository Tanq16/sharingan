package utils

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
)

var (
	infoStyle    = lipgloss.NewStyle().Foreground(ColorBlue)
	successStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	errorStyle   = lipgloss.NewStyle().Foreground(ColorRed)
	warnStyle    = lipgloss.NewStyle().Foreground(ColorYellow)
)

func PrintInfo(msg string) {
	printInfo("", msg)
}

func PrintSuccess(msg string) {
	printSuccess("", msg)
}

func PrintError(msg string, err error) {
	printError("", msg, err)
}

func PrintWarn(msg string, err error) {
	printWarn("", msg, err)
}

func PrintRunning(msg string) {
	printRunning("", msg)
}

func PrintIndentedSuccess(msg string) {
	printSuccess("  ", msg)
}

func PrintIndentedError(msg string, err error) {
	printError("  ", msg, err)
}

func PrintIndentedWarn(msg string, err error) {
	printWarn("  ", msg, err)
}

func PrintIndentedRunning(msg string) {
	printRunning("  ", msg)
}

func PrintFatal(msg string, err error) {
	printError("", msg, err)
	os.Exit(1)
}

func PrintGeneric(msg string) {
	lipgloss.Println(msg)
}

func printInfo(indent, msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
		return
	}
	lipgloss.Println(infoStyle.Render(indent + "→ " + msg))
}

func printSuccess(indent, msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
		return
	}
	lipgloss.Println(successStyle.Render(indent + "✓ " + msg))
}

func printError(indent, msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Error().Err(err).Msg(msg)
		} else {
			log.Error().Msg(msg)
		}
		return
	}
	lipgloss.Println(errorStyle.Render(indent + "✗ " + msg))
}

func printWarn(indent, msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Warn().Err(err).Msg(msg)
		} else {
			log.Warn().Msg(msg)
		}
		return
	}
	lipgloss.Println(warnStyle.Render(indent + "! " + msg))
}

func printRunning(indent, msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
		return
	}
	lipgloss.Println(infoStyle.Render(indent + "↻ " + msg))
}

func ClearLines(n int) {
	if GlobalDebugFlag || !StdoutIsTerminal {
		return
	}
	for range n {
		fmt.Print("\033[A\033[2K")
	}
}

func ClearPreviousLine() {
	ClearLines(1)
}

func PrintProgress(label string, percent int) {
	percent = min(max(percent, 0), 100)

	if GlobalDebugFlag {
		log.Info().Int("percent", percent).Msg(label)
		return
	}
	if !StdoutIsTerminal {
		lipgloss.Println(fmt.Sprintf("  ↻ %s: %d%%", label, percent))
		return
	}
	const barWidth = 10
	filled := barWidth * percent / 100
	bar := strings.Repeat("⣿", filled) + strings.Repeat("⣀", barWidth-filled)
	lipgloss.Println(infoStyle.Render(fmt.Sprintf("  ↻ %s: %s %d%%", label, bar, percent)))
}
