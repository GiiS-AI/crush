package dialog

import (
	"fmt"
	"image"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/question"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/common"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type ConfirmComponent struct {
	Styles           *styles.Styles
	Title            string
	Description      string
	QuestionLabels   []string
	QuestionRequests []question.Question
	Answers          []*question.Answer
	confirmYes       bool

	keyLeft  key.Binding
	keyRight key.Binding
	keyYes   key.Binding
	keyNo    key.Binding
	keyEnter key.Binding
	keyClose key.Binding
	keyUp    key.Binding
	keyDown  key.Binding

	focused      bool
	lastWidth    int
	scrollOffset int
	compositor   *lipgloss.Compositor
	hoverX       int
	hoverY       int

	OnConfirm func()
	OnReject  func()
}

func NewConfirmComponent(sty *styles.Styles, title, description string, labels []string, requests []question.Question, answers []*question.Answer) *ConfirmComponent {
	if title == "" || title == "Confirm" {
		title = "Ready to go?"
	}
	return &ConfirmComponent{
		Styles:           sty,
		Title:            title,
		Description:      description,
		QuestionLabels:   labels,
		QuestionRequests: requests,
		Answers:          answers,
		confirmYes:       true,
		keyLeft:          key.NewBinding(key.WithKeys("left"), key.WithHelp("←/→", "switch")),
		keyRight:         key.NewBinding(key.WithKeys("right"), key.WithHelp("←/→", "switch")),
		keyYes:           key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		keyNo:            key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "no")),
		keyEnter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		keyClose:         CloseKey,
		keyUp:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "scroll")),
		keyDown:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "scroll")),
	}
}

func (c *ConfirmComponent) HandleKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, c.keyUp):
		if c.scrollOffset > 0 {
			c.scrollOffset--
		}
		return false, nil
	case key.Matches(msg, c.keyDown):
		c.scrollOffset++
		return false, nil
	case key.Matches(msg, c.keyLeft), key.Matches(msg, c.keyRight):
		c.confirmYes = !c.confirmYes
		return false, nil
	case key.Matches(msg, c.keyYes):
		c.confirmYes = true
		return false, nil
	case key.Matches(msg, c.keyNo):
		c.confirmYes = false
		return false, nil
	case key.Matches(msg, c.keyEnter):
		if c.confirmYes {
			if c.OnConfirm != nil {
				c.OnConfirm()
			}
			return true, nil
		}
		if c.OnReject != nil {
			c.OnReject()
		}
		return false, nil
	case key.Matches(msg, c.keyClose):
		if c.OnReject != nil {
			c.OnReject()
		}
		return false, nil
	}
	return false, nil
}

func (c *ConfirmComponent) Response() question.Answer {
	return question.Answer{}
}

func (c *ConfirmComponent) ShortHelp() []key.Binding {
	return []key.Binding{c.keyUp, c.keyLeft, c.keyYes, c.keyNo, c.keyEnter, c.keyClose}
}

func (c *ConfirmComponent) unansweredCount() int {
	n := 0
	for _, ans := range c.Answers {
		if ans == nil || (len(ans.SelectedIDs) == 0 && ans.FillInText == "" && ans.Yes == nil) {
			n++
		}
	}
	return n
}

func (c *ConfirmComponent) Height(width int) int {
	w := width
	if w <= 0 {
		w = c.lastWidth
	}
	if w <= 0 {
		w = choiceListMaxWidth
	}
	iconPrompt := questionIconPrompt(c.Styles, c.focused)
	h := sectionHeight(c.Title, w-lipgloss.Width(iconPrompt))
	h++
	if c.Description != "" {
		r := common.MarkdownRenderer(c.Styles, w)
		mu := common.LockMarkdownRenderer(r)
		mu.Lock()
		out, err := r.Render(c.Description)
		mu.Unlock()
		if err == nil {
			out = strings.TrimSuffix(out, "\n")
			h += strings.Count(out, "\n") + 1
		} else {
			h += sectionHeight(c.Description, w)
		}
		h++
	}
	h += len(c.QuestionLabels)
	h++
	if c.unansweredCount() > 0 {
		h++
		h++
	}
	h++
	h++
	return h
}

func (c *ConfirmComponent) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	c.lastWidth = area.Dx()
	viewport := area.Dy()

	type line struct {
		text    string
		buttons bool
	}
	var lines []line

	iconPrompt := questionIconPrompt(c.Styles, c.focused)
	iconWidth := lipgloss.Width(iconPrompt)

	titleWrapped := ansi.Wrap(c.Title, area.Dx()-iconWidth, "")
	for _, l := range strings.Split(titleWrapped, "\n") {
		lines = append(lines, line{text: iconPrompt + c.Styles.Editor.QuestionUnselected.Render(l)})
	}
	lines = append(lines, line{})

	if c.Description != "" {
		r := common.MarkdownRenderer(c.Styles, area.Dx())
		mu := common.LockMarkdownRenderer(r)
		mu.Lock()
		desc, err := r.Render(c.Description)
		mu.Unlock()
		if err == nil {
			desc = strings.TrimSuffix(desc, "\n")
			for _, l := range strings.Split(desc, "\n") {
				lines = append(lines, line{text: l})
			}
		} else {
			for _, l := range strings.Split(c.Description, "\n") {
				lines = append(lines, line{text: l})
			}
		}
		lines = append(lines, line{})
	}

	bulletStyle := c.Styles.Editor.QuestionUnselected
	for i, label := range c.QuestionLabels {
		summary := c.answerSummary(i)
		bullet := bulletStyle.Render("• " + label + ": " + summary)
		lines = append(lines, line{text: bullet})
	}
	lines = append(lines, line{})

	if missed := c.unansweredCount(); missed > 0 {
		warnStyle := c.Styles.Tool.WarnTag
		msgStyle := c.Styles.Tool.WarnMessage
		word := "question"
		if missed > 1 {
			word = "questions"
		}
		warn := warnStyle.Render("WARN") + " " + msgStyle.Render(fmt.Sprintf("%d %s unanswered", missed, word))
		lines = append(lines, line{text: warn})
		lines = append(lines, line{})
	}

	lines = append(lines, line{buttons: true})
	lines = append(lines, line{})

	totalLines := len(lines)
	confirmButtonOpts := []common.ButtonOpts{
		{Text: "Yup!", Selected: c.confirmYes, Padding: 3, UnderlineIndex: 0},
		{Text: "Not yet", Selected: !c.confirmYes, Padding: 3, UnderlineIndex: 0},
	}
	overflow := viewport > 0 && totalLines > viewport

	maxScroll := totalLines - viewport
	if maxScroll < 0 {
		maxScroll = 0
	}
	if c.scrollOffset > maxScroll {
		c.scrollOffset = maxScroll
	}
	if c.scrollOffset < 0 {
		c.scrollOffset = 0
	}

	contentWidth := area.Dx()
	if overflow {
		contentWidth--
	}

	for screenRow := range viewport {
		idx := c.scrollOffset + screenRow
		if idx >= totalLines {
			break
		}
		ln := lines[idx]
		y := area.Min.Y + screenRow
		if ln.buttons {
			c.compositor = common.ButtonHitCompositor(c.Styles, confirmButtonOpts, " ", area.Min.X, y)
			hoveredBtn := common.HitButtonIndex(c.compositor, c.hoverX, c.hoverY)
			confirmButtonOpts[0].Hovered = hoveredBtn == 0
			confirmButtonOpts[1].Hovered = hoveredBtn == 1
			buttons := common.ButtonGroup(c.Styles, confirmButtonOpts, " ")
			drawStyledText(scr, image.Rect(area.Min.X, y, area.Min.X+contentWidth, y+1), buttons)
		} else if ln.text != "" {
			drawStyledText(scr, image.Rect(area.Min.X, y, area.Min.X+contentWidth, y+1), ln.text)
		}
	}

	if overflow {
		sb := common.Scrollbar(c.Styles, viewport, totalLines, viewport, c.scrollOffset)
		if sb != "" {
			x := area.Max.X - 1
			uv.NewStyledString(sb).Draw(scr, image.Rect(x, area.Min.Y, x+1, area.Min.Y+viewport))
		}
	}

	return nil
}

func (c *ConfirmComponent) HeightChanged() bool     { return false }
func (c *ConfirmComponent) SetFocused(focused bool) { c.focused = focused }
func (c *ConfirmComponent) SetHover(x, y int)       { c.hoverX = x; c.hoverY = y }

func (c *ConfirmComponent) HandleMouseClick(x, y int) (bool, bool) {
	switch common.HitButtonIndex(c.compositor, x, y) {
	case 0:
		c.confirmYes = true
		if c.OnConfirm != nil {
			c.OnConfirm()
		}
		return true, true
	case 1:
		c.confirmYes = false
		if c.OnReject != nil {
			c.OnReject()
		}
		return false, true
	}
	return false, false
}

func (c *ConfirmComponent) UpdateAnswers(answers []*question.Answer) {
	c.Answers = answers
}

func (c *ConfirmComponent) answerSummary(idx int) string {
	if idx >= len(c.Answers) || c.Answers[idx] == nil {
		return "(not answered)"
	}
	resp := c.Answers[idx]
	var parts []string
	if len(resp.SelectedIDs) > 0 {
		labels := make([]string, 0, len(resp.SelectedIDs))
		for _, id := range resp.SelectedIDs {
			labels = append(labels, c.choiceLabel(idx, id))
		}
		parts = append(parts, strings.Join(labels, ", "))
	}
	if resp.FillInText != "" {
		parts = append(parts, resp.FillInText)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "; ")
	}
	if resp.Yes != nil {
		if *resp.Yes {
			return "Yes"
		}
		return "No"
	}
	return "(not answered)"
}

func (c *ConfirmComponent) choiceLabel(qIdx int, choiceID string) string {
	if qIdx < len(c.QuestionRequests) {
		for _, ch := range c.QuestionRequests[qIdx].Choices {
			if ch.ID == choiceID {
				return ch.Label
			}
		}
	}
	return choiceID
}
