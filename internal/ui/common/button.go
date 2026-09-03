package common

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/styles"
)

// ButtonOpts defines the configuration for a single button
type ButtonOpts struct {
	// Text is the button label
	Text string
	// UnderlineIndex is the 0-based index of the character to underline (-1 for none)
	UnderlineIndex int
	// Selected indicates whether this button is currently selected
	Selected bool
	// Hovered indicates whether the mouse is hovering over the button.
	Hovered bool
	// Padding inner horizontal padding defaults to 2 if this is 0
	Padding int
}

// Button creates a button with an underlined character and selection state
func Button(t *styles.Styles, opts ButtonOpts) string {
	// Select style based on selection and hover state.
	style := t.Button.Blurred
	if opts.Selected && opts.Hovered {
		style = t.Button.Focused.Bold(true)
	} else if opts.Hovered {
		style = t.Button.Hovered.Bold(true)
	} else if opts.Selected {
		style = t.Button.Focused
	}

	text := opts.Text
	if opts.Padding == 0 {
		opts.Padding = 2
	}

	// the index is out of bound
	if opts.UnderlineIndex > -1 && opts.UnderlineIndex > len(text)-1 {
		opts.UnderlineIndex = -1
	}

	text = style.Padding(0, opts.Padding).Render(text)

	if opts.UnderlineIndex != -1 {
		text = lipgloss.StyleRanges(text, lipgloss.NewRange(opts.Padding+opts.UnderlineIndex, opts.Padding+opts.UnderlineIndex+1, style.Underline(true)))
	}

	return text
}

// ButtonGroup creates a row of selectable buttons
// Spacing is the separator between buttons
// Use "  " or similar for horizontal layout
// Use "\n"  for vertical layout
// Defaults to "  " (horizontal)
func ButtonGroup(t *styles.Styles, buttons []ButtonOpts, spacing string) string {
	if len(buttons) == 0 {
		return ""
	}

	if spacing == "" {
		spacing = "  "
	}

	parts := make([]string, len(buttons))
	for i, button := range buttons {
		parts[i] = Button(t, button)
	}

	return strings.Join(parts, spacing)
}

// ButtonHitCompositor builds a compositor with one hit layer per button.
func ButtonHitCompositor(sty *styles.Styles, opts []ButtonOpts, spacing string, x, y int) *lipgloss.Compositor {
	if len(opts) == 0 {
		return nil
	}
	if spacing == "" {
		spacing = "  "
	}
	spacingWidth := lipgloss.Width(spacing)
	var layers []*lipgloss.Layer
	buttonX := x
	for i, opt := range opts {
		button := Button(sty, opt)
		width := lipgloss.Width(button)
		layers = append(layers, lipgloss.NewLayer(strings.Repeat(" ", width)).X(buttonX).Y(y).ID(fmt.Sprintf("btn_%d", i)))
		buttonX += width + spacingWidth
	}
	return lipgloss.NewCompositor(layers...)
}

// HitButtonIndex returns the hit button index or -1 when nothing was hit.
func HitButtonIndex(c *lipgloss.Compositor, x, y int) int {
	if c == nil {
		return -1
	}
	hit := c.Hit(x, y)
	if hit.Empty() {
		return -1
	}
	var idx int
	if _, err := fmt.Sscanf(hit.ID(), "btn_%d", &idx); err != nil {
		return -1
	}
	return idx
}
