package model

import (
	"image"

	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/logo"
	"github.com/GiiS-AI/GiiS-Code/internal/version"
	"github.com/GiiS-AI/GiiS-Code/internal/workspace"
	"github.com/charmbracelet/ultraviolet/layout"
)

const (
	landingLogoColumnWidth  = 22
	landingDashboardGap     = 12
	landingMinWideWidth     = 84
	landingResourceMaxWidth = 14
	landingSkillsMaxWidth   = 24
	landingStackedMaxWidth  = 20
	landingWideLogoTopPad   = 2
)

// selectedLargeModel returns the currently selected large language model from
// the agent coordinator, if one exists.
func (m *UI) selectedLargeModel() *workspace.AgentModel {
	if m.com.Workspace.AgentIsReady() {
		model := m.com.Workspace.AgentModel()
		return &model
	}
	return nil
}

// landingView renders the landing page view showing the current working
// directory, model information, and LSP/MCP/skills status. Wide terminals keep
// the wordmark anchored on the left and treat the session details as a compact
// top dashboard; narrow terminals fall back to the original stacked layout.
func (m *UI) landingView() string {
	width := m.layout.main.Dx()

	content := m.landingDashboard(width)
	return lipgloss.NewStyle().
		Width(width).
		Height(m.layout.main.Dy() - 1).
		PaddingTop(1).
		Render(content)
}

func (m *UI) landingDashboard(width int) string {
	if width < landingMinWideWidth {
		return m.landingStacked(width)
	}

	t := m.com.Styles
	logoColumnWidth := min(landingLogoColumnWidth, max(12, width/4))
	availableDashboardWidth := width - logoColumnWidth
	dashboardWidth := min(98, availableDashboardWidth-landingDashboardGap)
	if dashboardWidth < 48 {
		return m.landingStacked(width)
	}

	wordmark := logo.Render(t.Logo.GradCanvas, version.Version, true, logo.Opts{
		FieldColor:   t.Logo.FieldColor,
		TitleColorA:  t.Logo.TitleColorA,
		TitleColorB:  t.Logo.TitleColorB,
		CharmColor:   t.Logo.CharmColor,
		VersionColor: t.Logo.VersionColor,
		Width:        logoColumnWidth,
	})

	infoSection := lipgloss.JoinVertical(
		lipgloss.Left,
		m.modelInfo(dashboardWidth),
	)

	var remainingHeightArea image.Rectangle
	layout.Vertical(
		layout.Len(lipgloss.Height(infoSection)+1),
		layout.Fill(1),
	).Split(m.layout.main).Assign(new(image.Rectangle), &remainingHeightArea)

	sectionWidth := min(landingResourceMaxWidth, max(10, (dashboardWidth-landingSkillsMaxWidth-2)/2))
	skillsWidth := min(landingSkillsMaxWidth, max(18, dashboardWidth-(sectionWidth*2)-2))
	maxItemsPerSection := max(1, remainingHeightArea.Dy())
	lspSection := m.lspInfo(sectionWidth, maxItemsPerSection, false)
	mcpSection := m.mcpInfo(sectionWidth, maxItemsPerSection, false)
	skillsSection := m.skillsInfo(skillsWidth, maxItemsPerSection, false)
	resources := lipgloss.NewStyle().
		Width(dashboardWidth).
		Align(lipgloss.Right).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, lspSection, " ", mcpSection, " ", skillsSection))

	dashboard := lipgloss.NewStyle().
		Width(dashboardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, infoSection, resources))
	spacerWidth := max(landingDashboardGap, availableDashboardWidth-dashboardWidth)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(logoColumnWidth).PaddingTop(landingWideLogoTopPad).Render(wordmark),
		lipgloss.NewStyle().Width(spacerWidth).Render(""),
		dashboard,
	)
}

func (m *UI) landingStacked(width int) string {
	t := m.com.Styles
	wordmark := logo.Render(t.Logo.GradCanvas, version.Version, true, logo.Opts{
		FieldColor:   t.Logo.FieldColor,
		TitleColorA:  t.Logo.TitleColorA,
		TitleColorB:  t.Logo.TitleColorB,
		CharmColor:   t.Logo.CharmColor,
		VersionColor: t.Logo.VersionColor,
		Width:        width,
	})

	parts := []string{wordmark, "", m.modelInfo(width)}
	infoSection := lipgloss.JoinVertical(lipgloss.Left, parts...)

	var remainingHeightArea image.Rectangle
	layout.Vertical(
		layout.Len(lipgloss.Height(infoSection)+1),
		layout.Fill(1),
	).Split(m.layout.main).Assign(new(image.Rectangle), &remainingHeightArea)

	mcpLspSectionWidth := min(landingStackedMaxWidth, (width-2)/3)
	lspSection := m.lspInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
	mcpSection := m.mcpInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
	skillsSection := m.skillsInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
	content := lipgloss.JoinHorizontal(lipgloss.Left, lspSection, " ", mcpSection, " ", skillsSection)

	return lipgloss.JoinVertical(lipgloss.Left, infoSection, "", content)
}
