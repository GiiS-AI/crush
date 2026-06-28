// Package logo renders a GiiS-Code wordmark in a stylized way.
package logo

import (
	_ "embed"
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/styles"
)

// letterform represents a letterform. It can be stretched horizontally by
// a given amount via the boolean argument.
type letterform func(bool) string

const diag = `╱`

// Opts are the options for rendering the GiiS-Code title art.
type Opts struct {
	FieldColor   color.Color // diagonal lines
	TitleColorA  color.Color // left gradient ramp point
	TitleColorB  color.Color // right gradient ramp point
	CharmColor   color.Color // Charm™ text color
	VersionColor color.Color // version text color
	Width        int         // width of the rendered logo, used for truncation
	Hyper        bool        // reserved for future GiiS-Code Pro mode

	// When true, stretch a random letterform on each render. Has no effect in
	// compact mode. Mainly for testing. In production you will want to cache
	// the stretched letterform to keep the logo from jittering on resize.
	Unstable bool
}

//go:embed asciigiis.txt
var giisLogo string

// Render renders the GiiS-Code logo. Set the argument to true to render the
// narrow version, intended for use in a sidebar.
//
// The compact argument determines whether it renders compact for the sidebar
// or wider for the main pane.
func Render(base lipgloss.Style, version string, compact bool, o Opts) string {
	if o.TitleColorA == nil || o.TitleColorB == nil {
		return strings.TrimRight(giisLogo, "\n")
	}

	lines := strings.Split(strings.TrimRight(giisLogo, "\n"), "\n")
	colored := make([]string, len(lines))
	for i, line := range lines {
		colored[i] = styles.ApplyBoldForegroundGrad(base, line, o.TitleColorA, o.TitleColorB)
	}
	result := strings.Join(colored, "\n")

	if version != "" {
		versionStyle := base.Foreground(o.VersionColor)
		result += "\n" + versionStyle.Render("  v"+version)
	}

	return result
}

// SmallRender renders a smaller version of the GiiS-Code logo, suitable for
// smaller windows or sidebar usage.
func SmallRender(t *styles.Styles, width int, o Opts) string {
	name := "GiiS-Code"
	brand := " GiiS AI"
	title := t.Logo.SmallCharm.Render(brand)
	title = fmt.Sprintf("%s %s", title, styles.ApplyBoldForegroundGrad(t.Logo.GradCanvas, name, t.Logo.SmallGradFromColor, t.Logo.SmallGradToColor))
	remainingWidth := width - lipgloss.Width(title) - 1 // 1 for the space after the name
	if remainingWidth > 0 {
		lines := strings.Repeat("╱", remainingWidth)
		title = fmt.Sprintf("%s %s", title, t.Logo.SmallDiagonals.Render(lines))
	}
	return title
}
