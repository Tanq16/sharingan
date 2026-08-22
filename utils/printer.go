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
	fmt.Println(msg)
}

func printInfo(indent, msg string) {
	switch {
	case GlobalDebugFlag:
		log.Info().Msg(msg)
	case GlobalForAIFlag:
		fmt.Println("[INFO] " + msg)
	default:
		fmt.Println(infoStyle.Render(indent + "→ " + msg))
	}
}

func printSuccess(indent, msg string) {
	switch {
	case GlobalDebugFlag:
		log.Info().Msg(msg)
	case GlobalForAIFlag:
		fmt.Println("[OK] " + msg)
	default:
		fmt.Println(successStyle.Render(indent + "✓ " + msg))
	}
}

func printError(indent, msg string, err error) {
	switch {
	case GlobalDebugFlag:
		if err != nil {
			log.Error().Err(err).Msg(msg)
		} else {
			log.Error().Msg(msg)
		}
	case GlobalForAIFlag:
		fmt.Println("[ERROR] " + msg)
	default:
		fmt.Println(errorStyle.Render(indent + "✗ " + msg))
	}
}

func printWarn(indent, msg string, err error) {
	switch {
	case GlobalDebugFlag:
		if err != nil {
			log.Warn().Err(err).Msg(msg)
		} else {
			log.Warn().Msg(msg)
		}
	case GlobalForAIFlag:
		fmt.Println("[WARN] " + msg)
	default:
		fmt.Println(warnStyle.Render(indent + "! " + msg))
	}
}

func printRunning(indent, msg string) {
	switch {
	case GlobalDebugFlag:
		log.Info().Msg(msg)
	case GlobalForAIFlag:
		fmt.Println("[RUNNING] " + msg)
	default:
		fmt.Println(infoStyle.Render(indent + "↻ " + msg))
	}
}

// Clearing is a no-op in debug and AI modes so every line survives being logged or parsed.
func ClearLines(n int) {
	if GlobalDebugFlag || GlobalForAIFlag {
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

	switch {
	case GlobalDebugFlag:
		log.Info().Int("percent", percent).Msg(label)
	case GlobalForAIFlag:
		fmt.Printf("[PROGRESS] %s: %d%%\n", label, percent)
	default:
		const barWidth = 10
		filled := barWidth * percent / 100
		bar := strings.Repeat("⣿", filled) + strings.Repeat("⣀", barWidth-filled)
		fmt.Println(infoStyle.Render(fmt.Sprintf("  ↻ %s: %s %d%%", label, bar, percent)))
	}
}
