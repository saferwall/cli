// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	fieldURL = iota
	fieldUsername
	fieldPassword
	fieldCount
)

type initModel struct {
	inputs    []textinput.Model
	focused   int
	submitted bool
}

func newInitModel() initModel {
	inputs := make([]textinput.Model, fieldCount)

	url := textinput.New()
	url.Placeholder = "https://api.saferwall.com"
	url.SetValue("https://api.saferwall.com")
	url.Focus()
	url.Prompt = "  "
	inputs[fieldURL] = url

	username := textinput.New()
	username.Placeholder = "your username"
	username.Prompt = "  "
	inputs[fieldUsername] = username

	password := textinput.New()
	password.Placeholder = "your password"
	password.EchoMode = textinput.EchoPassword
	password.Prompt = "  "
	inputs[fieldPassword] = password

	return initModel{inputs: inputs}
}

func (m initModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "down":
			m.focused = (m.focused + 1) % fieldCount
			return m, m.updateFocus()
		case "shift+tab", "up":
			m.focused = (m.focused - 1 + fieldCount) % fieldCount
			return m, m.updateFocus()
		case "enter":
			if m.focused == fieldPassword {
				m.submitted = true
				return m, tea.Quit
			}
			m.focused++
			return m, m.updateFocus()
		}
	}

	// Update the focused input.
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *initModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focused {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m initModel) View() string {
	if m.submitted {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(styleLabel.Render("  Configure Saferwall CLI") + "\n\n")

	labels := []string{"URL", "Username", "Password"}
	for i, label := range labels {
		if i == m.focused {
			b.WriteString(styleSuccess.Render("  > "))
		} else {
			b.WriteString("    ")
		}
		b.WriteString(fmt.Sprintf("%-10s", label))
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleDim.Render("  tab/shift+tab: navigate  enter: confirm  esc: cancel"))
	b.WriteString("\n")
	return b.String()
}

func (m initModel) values() (url, username, password string) {
	return m.inputs[fieldURL].Value(),
		m.inputs[fieldUsername].Value(),
		m.inputs[fieldPassword].Value()
}
