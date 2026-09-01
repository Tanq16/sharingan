package utils

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

var ErrNoTerminal = errors.New("no interactive terminal")

type inputModel struct {
	textInput textinput.Model
	done      bool
	value     string
	initCmd   tea.Cmd
}

func (m inputModel) Init() tea.Cmd { return m.initCmd }

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			m.value = m.textInput.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textInput.View())
}

func PromptInput(prompt, placeholder string) (string, error) {
	if !StdinIsTerminal {
		return "", ErrNoTerminal
	}
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = prompt + " "
	m := inputModel{textInput: ti, initCmd: ti.Focus()}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(final.(inputModel).value), nil
}

// A password keeps its surrounding whitespace, which PromptInput trims away.
func PromptPassword(prompt string) (string, error) {
	if !StdinIsTerminal {
		return "", ErrNoTerminal
	}
	ti := textinput.New()
	ti.Placeholder = "••••••••"
	ti.Prompt = prompt + " "
	ti.EchoMode = textinput.EchoPassword
	m := inputModel{textInput: ti, initCmd: ti.Focus()}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", err
	}
	return final.(inputModel).value, nil
}

type textAreaModel struct {
	textarea textarea.Model
	done     bool
	value    string
	initCmd  tea.Cmd
}

func (m textAreaModel) Init() tea.Cmd { return m.initCmd }

func (m textAreaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+d":
			m.value = m.textarea.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m textAreaModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textarea.View() + "\n(Ctrl+D to submit, Esc to cancel)")
}

func PromptTextArea(prompt, placeholder string) (string, error) {
	if !StdinIsTerminal {
		return "", ErrNoTerminal
	}
	PrintInfo(prompt)

	ta := textarea.New()
	ta.Placeholder = placeholder
	m := textAreaModel{textarea: ta, initCmd: ta.Focus()}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(final.(textAreaModel).value), nil
}

type selectModel struct {
	label    string
	options  []string
	cursor   int
	chosen   map[int]bool
	multi    bool
	canceled bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = min(m.cursor+1, len(m.options)-1)
	case " ":
		if m.multi {
			m.chosen[m.cursor] = !m.chosen[m.cursor]
		}
	case "enter":
		return m, tea.Quit
	case "esc", "ctrl+c":
		m.canceled = true
		return m, tea.Quit
	}
	return m, nil
}

func (m selectModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.label + "\n")
	for i, option := range m.options {
		line := "  "
		if i == m.cursor {
			line = "> "
		}
		if m.multi {
			if m.chosen[i] {
				line += "[x] "
			} else {
				line += "[ ] "
			}
		}
		line += option
		if i == m.cursor {
			line = infoStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return tea.NewView(b.String())
}

func PromptSelect(label string, options []string) (int, error) {
	if !StdinIsTerminal {
		return -1, ErrNoTerminal
	}
	final, err := tea.NewProgram(selectModel{label: label, options: options}).Run()
	if err != nil {
		return -1, err
	}
	ClearLines(len(options) + 1)
	m := final.(selectModel)
	if m.canceled {
		return -1, nil
	}
	return m.cursor, nil
}

func PromptMultiSelect(label string, options []string) (map[int]bool, error) {
	if !StdinIsTerminal {
		return nil, ErrNoTerminal
	}
	final, err := tea.NewProgram(selectModel{
		label: label, options: options, chosen: map[int]bool{}, multi: true,
	}).Run()
	if err != nil {
		return nil, err
	}
	ClearLines(len(options) + 1)
	m := final.(selectModel)
	if m.canceled {
		return nil, nil
	}
	return m.chosen, nil
}
