package model

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/common"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/util"
)

// desktopAppBinary is the executable name GiiS Desktop installs on PATH.
const desktopAppBinary = "giis-desktop"

// updateDesktopLaunchPromptView handles keyboard input for the GiiS Desktop
// auto-launch prompt shown on first run.
func (m *UI) updateDesktopLaunchPromptView(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.DesktopLaunch.Enter):
		cmds = append(cmds, m.confirmDesktopLaunchChoice())
	case key.Matches(msg, m.keyMap.DesktopLaunch.Switch):
		m.desktopLaunch.yesSelected = !m.desktopLaunch.yesSelected
	case key.Matches(msg, m.keyMap.DesktopLaunch.Yes):
		m.desktopLaunch.yesSelected = true
		cmds = append(cmds, m.confirmDesktopLaunchChoice())
	case key.Matches(msg, m.keyMap.DesktopLaunch.No):
		m.desktopLaunch.yesSelected = false
		cmds = append(cmds, m.confirmDesktopLaunchChoice())
	case key.Matches(msg, m.keyMap.DesktopLaunch.ToggleRemember):
		m.desktopLaunch.rememberChoice = !m.desktopLaunch.rememberChoice
	}
	return cmds
}

// confirmDesktopLaunchChoice persists the choice (if "remember" is checked),
// launches GiiS Desktop (if selected), and transitions to the landing view.
func (m *UI) confirmDesktopLaunchChoice() tea.Cmd {
	var cmds []tea.Cmd

	if m.desktopLaunch.rememberChoice {
		choice := m.desktopLaunch.yesSelected
		if err := m.com.Workspace.SetConfigField(
			config.ScopeGlobal, "options.desktop_auto_launch", choice,
		); err != nil {
			cmds = append(cmds, util.ReportError(err))
		}
	}

	if m.desktopLaunch.yesSelected {
		if err := m.launchDesktopApp(); err != nil {
			cmds = append(cmds, util.ReportError(fmt.Errorf("couldn't launch %s: %w", desktopAppBinary, err)))
		}
	}

	m.setState(uiLanding, uiFocusEditor)
	return tea.Batch(cmds...)
}

// launchDesktopApp starts the GiiS Desktop app as a fully detached process
// so it survives independently of c0d3r's own lifetime and never blocks
// the TUI. Unlike openEditor's tea.ExecProcess (which hands over the
// terminal and waits), this is a plain background spawn.
func (m *UI) launchDesktopApp() error {
	binPath, err := exec.LookPath(desktopAppBinary)
	if err != nil {
		return err
	}

	cmd := exec.Command(binPath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the child in the background instead of leaving a zombie; we
	// never wait on it to block, just to release resources once it exits.
	go func() { _ = cmd.Wait() }()
	return nil
}

// desktopLaunchPromptView renders the GiiS Desktop auto-launch prompt.
func (m *UI) desktopLaunchPromptView() string {
	s := m.com.Styles.Initialize

	header := s.Header.Render("Launch GiiS Desktop automatically?")
	desc := s.Content.Render("GiiS Desktop connects your installed Claude Code / Codex CLIs to GiiS Chat. You can launch it manually anytime by running " + desktopAppBinary + ".")

	rememberLabel := "[ ] Remember my choice"
	if m.desktopLaunch.rememberChoice {
		rememberLabel = "[x] Remember my choice"
	}
	remember := s.Content.Render(rememberLabel + " (ctrl+r)")

	buttons := common.ButtonGroup(m.com.Styles, []common.ButtonOpts{
		{Text: "Yes", Selected: m.desktopLaunch.yesSelected},
		{Text: "No", Selected: !m.desktopLaunch.yesSelected},
	}, " ")

	hint := s.Content.Render("enter/ctrl+d to confirm, tab/←/→ to switch")

	lines := []string{header, desc, remember, buttons, hint}

	width := min(m.layout.main.Dx(), 60)

	return lipgloss.NewStyle().
		Width(width).
		Height(m.layout.main.Dy()).
		PaddingBottom(1).
		AlignVertical(lipgloss.Bottom).
		Render(strings.Join(lines, "\n\n"))
}
