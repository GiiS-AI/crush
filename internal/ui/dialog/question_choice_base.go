package dialog

import (
	"fmt"
	"image"
	"strconv"
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

const choiceListMaxWidth = 120

func questionIconPrompt(sty *styles.Styles, focused bool) string {
	if focused {
		return sty.Editor.PromptQuestionIconFocused.Render()
	}
	return sty.Editor.PromptQuestionIconBlurred.Render()
}

type choiceList struct {
	questionEditor
	Request question.Question

	cursorIdx        int
	scrollOffset     int
	focused          bool
	lastWidth        int
	choiceCompositor *lipgloss.Compositor
	suppressScroll   bool
	wheelActive      bool
	hoverX, hoverY   int
	hoveredChoice    int
	mouseActive      bool

	lastLines    []contentLine
	lastViewport int
	fillInTop    int
	fillInBottom int

	keyUp    key.Binding
	keyDown  key.Binding
	keyClose key.Binding
}

func (c *choiceList) numberKeyIndex(msg tea.KeyPressMsg) int {
	if len(msg.Text) != 1 {
		return -1
	}
	n, err := strconv.Atoi(msg.Text)
	if err != nil || n < 1 || n > len(c.Request.Choices) {
		return -1
	}
	return n - 1
}

func newChoiceList(sty *styles.Styles, req question.Question) choiceList {
	return choiceList{
		questionEditor: newQuestionEditor(sty),
		Request:        req,
		hoveredChoice:  -1,
		hoverX:         -1,
		hoverY:         -1,
		fillInTop:      -1,
		fillInBottom:   -1,
		keyUp:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
		keyDown:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
		keyClose:       CloseKey,
	}
}

func (c *choiceList) itemCount() int {
	return len(c.Request.Choices) + 1
}

func (c *choiceList) isFillIn() bool {
	return c.cursorIdx == len(c.Request.Choices)
}

func (c *choiceList) moveUp() {
	c.wheelActive = false
	if c.mouseActive {
		c.cursorIdx = max(c.hoveredChoice, 0)
	}
	c.mouseActive = false
	c.fillIn.Blur()
	if c.activeNoteKey != "" {
		c.closeNote(c.noteKey())
	}
	c.cursorIdx--
	if c.cursorIdx < 0 {
		c.cursorIdx = c.itemCount() - 1
	}
}

func (c *choiceList) moveDown() {
	c.wheelActive = false
	if c.mouseActive {
		c.cursorIdx = c.hoveredChoice
	}
	c.mouseActive = false
	c.fillIn.Blur()
	if c.activeNoteKey != "" {
		c.closeNote(c.noteKey())
	}
	c.cursorIdx++
	if c.cursorIdx >= c.itemCount() {
		c.cursorIdx = 0
	}
}

func (c *choiceList) adoptHover() {
	if c.mouseActive && c.hoveredChoice >= 0 {
		c.cursorIdx = c.hoveredChoice
	}
	c.mouseActive = false
}

func (c *choiceList) handleFillInKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, c.keyClose):
		c.fillIn.Blur()
		return nil, true
	case key.Matches(msg, c.navUp):
		c.mouseActive = false
		c.moveUp()
		if c.isFillIn() {
			c.fillIn.Focus()
			return c.fillIn.Focus(), true
		}
		return nil, true
	case key.Matches(msg, c.navDown):
		c.mouseActive = false
		c.moveDown()
		if c.isFillIn() {
			c.fillIn.Focus()
			return c.fillIn.Focus(), true
		}
		return nil, true
	default:
		c.wheelActive = false
		c.mouseActive = false
		var cmd tea.Cmd
		c.fillIn, cmd = c.fillIn.Update(msg)
		return cmd, true
	}
}

func (c *choiceList) handleNavKey(msg tea.KeyPressMsg) bool {
	switch {
	case key.Matches(msg, c.keyUp):
		c.moveUp()
		if c.isFillIn() {
			c.fillIn.Focus()
		}
		return true
	case key.Matches(msg, c.keyDown):
		c.moveDown()
		if c.isFillIn() {
			c.fillIn.Focus()
		}
		return true
	}
	return false
}

func (c *choiceList) noteKey() string {
	if c.isFillIn() || c.cursorIdx >= len(c.Request.Choices) {
		return "_question"
	}
	return c.Request.Choices[c.cursorIdx].ID
}

type contentLine struct {
	text       string
	fillInRow  bool
	noteRow    bool
	cursorItem bool
	choiceIdx  int
}

func newContentLine(text string) contentLine {
	return contentLine{text: text, choiceIdx: -1}
}

func sectionHeight(text string, width int) int {
	if text == "" {
		return 0
	}
	return strings.Count(ansi.Wrap(text, width, ""), "\n") + 1
}

func wrapIndent(text string, width int, indent string) string {
	wrapped := ansi.Wrap(text, width, "")
	lines := strings.Split(wrapped, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

func drawStyledText(scr uv.Screen, area uv.Rectangle, text string) int {
	if text == "" {
		return 0
	}
	uv.NewStyledString(text).Draw(scr, area)
	return strings.Count(text, "\n") + 1
}

func (c *choiceList) buildLines(innerWidth int, fillInPrefix string, itemFn choiceItemRenderer) []contentLine {
	bodyStyle := c.Styles.Editor.QuestionBody
	barActive := c.Styles.Editor.QuestionCursorBar.Render("┃ ")
	const barInactive = "  "

	var lines []contentLine
	push := func(text string, flags ...bool) {
		cl := newContentLine(text)
		if len(flags) > 0 {
			cl.fillInRow = flags[0]
		}
		if len(flags) > 1 {
			cl.cursorItem = flags[1]
		}
		for ln := range strings.SplitSeq(text, "\n") {
			row := cl
			row.text = ln
			lines = append(lines, row)
		}
	}

	icon := c.iconPrompt()
	iconWidth := lipgloss.Width(icon)
	qIndent := strings.Repeat(" ", iconWidth)
	push(icon + c.Styles.Editor.QuestionUnselected.Render(wrapIndent(c.Request.Text, innerWidth-iconWidth, qIndent)))
	push("")

	if c.Request.Description != "" {
		push(c.renderDescription(innerWidth))
		push("")
	}

	for i, ch := range c.Request.Choices {
		active := i == c.cursorIdx && !c.mouseActive
		hovered := i == c.hoveredChoice && c.mouseActive
		bar := barInactive
		if active || hovered {
			bar = barActive
		}
		content := itemFn(i, ch, active, innerWidth)
		for j, ln := range strings.Split(content, "\n") {
			b := bar
			if j > 0 && !active {
				b = barInactive
			}
			lines = append(lines, contentLine{text: b + ln, cursorItem: active, choiceIdx: i})
		}

		if ch.Description != "" {
			descContent := bodyStyle.Render(wrapIndent(ch.Description, innerWidth-lipgloss.Width(bar), ""))
			for j, ln := range strings.Split(descContent, "\n") {
				b := bar
				if j > 0 && !active {
					b = barInactive
				}
				lines = append(lines, contentLine{text: b + ln, cursorItem: active, choiceIdx: i})
			}
		}

		c.drawNote(&lines, innerWidth, bar, barInactive, ch.ID, active)
		lines = append(lines, contentLine{text: "", choiceIdx: i})
	}

	fillInIdx := len(c.Request.Choices)
	fillActive := c.isFillIn() && !c.mouseActive
	fillHovered := c.mouseActive && c.hoveredChoice == fillInIdx
	fillBar := barInactive
	if fillActive || fillHovered {
		fillBar = barActive
	}
	linesBeforeFillIn := len(lines)
	c.drawFillIn(&lines, innerWidth, fillBar, barInactive, fillInPrefix, c.isFillIn(), false)

	c.fillInTop = -1
	c.fillInBottom = -1
	for i := linesBeforeFillIn; i < len(lines); i++ {
		if c.fillInTop < 0 {
			c.fillInTop = i
		}
		c.fillInBottom = i
	}
	for i := linesBeforeFillIn; i < len(lines); i++ {
		lines[i].choiceIdx = fillInIdx
	}

	push("")
	return lines
}

func (c *choiceList) renderDescription(width int) string {
	r := common.MarkdownRenderer(c.Styles, width)
	mu := common.LockMarkdownRenderer(r)
	mu.Lock()
	out, err := r.Render(c.Request.Description)
	mu.Unlock()
	if err != nil {
		return c.Request.Description
	}
	return strings.TrimSuffix(out, "\n")
}

type choiceItemRenderer func(index int, choice question.Choice, active bool, innerWidth int) string

func (c *choiceList) height(width int) int {
	if width <= 0 {
		width = c.lastWidth
	}
	innerWidth := min(width-4, choiceListMaxWidth)
	return len(c.buildLines(innerWidth, "> ", func(int, question.Choice, bool, int) string {
		return "x"
	}))
}

func (c *choiceList) heightChanged() bool {
	return false
}

func (c *choiceList) setFocused(focused bool) {
	c.focused = focused
}

func (c *choiceList) setHover(x, y int) {
	c.hoverX = x
	c.hoverY = y
	c.mouseActive = true
	c.hoveredChoice = -1
	if c.choiceCompositor == nil {
		return
	}
	hit := c.choiceCompositor.Hit(x, y)
	if !hit.Empty() {
		var idx int
		if _, err := fmt.Sscanf(hit.ID(), "choice_%d", &idx); err == nil {
			c.hoveredChoice = idx
		}
	}
}

func (c *choiceList) iconPrompt() string {
	return questionIconPrompt(c.Styles, c.focused)
}

func (c *choiceList) drawContent(scr uv.Screen, area uv.Rectangle, fillInPrefix string, itemFn choiceItemRenderer) *tea.Cursor {
	c.lastWidth = area.Dx()
	viewport := area.Dy()

	contentWidth := area.Dx()
	innerNarrow := min(contentWidth-1-4, choiceListMaxWidth)
	innerWide := min(contentWidth-4, choiceListMaxWidth)

	lines := c.buildLines(innerWide, fillInPrefix, itemFn)
	overflow := viewport > 0 && len(lines) > viewport
	if overflow && innerNarrow != innerWide {
		lines = c.buildLines(innerNarrow, fillInPrefix, itemFn)
	}

	if overflow {
		contentWidth--
	}
	c.lastLines = lines
	c.lastViewport = viewport
	c.clampScroll(lines, viewport)

	var cur *tea.Cursor
	for screenRow := range viewport {
		idx := c.scrollOffset + screenRow
		if idx >= len(lines) {
			break
		}
		ln := lines[idx]
		y := area.Min.Y + screenRow
		if ln.text != "" {
			uv.NewStyledString(ln.text).Draw(scr, image.Rect(area.Min.X, y, area.Min.X+contentWidth, y+1))
		}
		if ln.fillInRow {
			fillPrefix := c.Styles.Editor.QuestionBody.Render("> ")
			if tc := c.fillInCursor(screenRow, area.Min.X, lipgloss.Width(fillPrefix)); tc != nil {
				cur = tc
			}
		}
		if ln.noteRow {
			const notePrefix = "> "
			if tc := c.noteCursor(screenRow, area.Min.X, lipgloss.Width(notePrefix)); tc != nil {
				cur = tc
			}
		}
	}

	if cur != nil {
		if cur.Y < 0 {
			cur.Y = 0
		} else if cur.Y >= viewport {
			cur.Y = viewport - 1
		}
		if cur.X < 0 {
			cur.X = 0
		} else if cur.X >= area.Dx() {
			cur.X = area.Dx() - 1
		}
	}

	if overflow {
		sb := common.Scrollbar(c.Styles, viewport, len(lines), viewport, c.scrollOffset)
		if sb != "" {
			x := area.Max.X - 1
			uv.NewStyledString(sb).Draw(scr, image.Rect(x, area.Min.Y, x+1, area.Min.Y+viewport))
		}
	}

	c.buildChoiceCompositor(lines, area, contentWidth)
	return cur
}

func (c *choiceList) buildChoiceCompositor(lines []contentLine, area uv.Rectangle, contentWidth int) {
	type rowRange struct{ min, max int }
	ranges := make(map[int]*rowRange)
	for screenRow := range area.Dy() {
		idx := c.scrollOffset + screenRow
		if idx >= len(lines) {
			break
		}
		ln := lines[idx]
		if ln.choiceIdx < 0 {
			continue
		}
		r, ok := ranges[ln.choiceIdx]
		if !ok {
			r = &rowRange{min: screenRow, max: screenRow}
			ranges[ln.choiceIdx] = r
		} else {
			if screenRow < r.min {
				r.min = screenRow
			}
			if screenRow > r.max {
				r.max = screenRow
			}
		}
	}

	var layers []*lipgloss.Layer
	for choiceIdx, r := range ranges {
		height := r.max - r.min + 1
		hitStr := strings.Repeat(strings.Repeat(" ", contentWidth)+"\n", height-1) + strings.Repeat(" ", contentWidth)
		y := area.Min.Y + r.min
		layers = append(layers, lipgloss.NewLayer(hitStr).X(area.Min.X).Y(y).ID(fmt.Sprintf("choice_%d", choiceIdx)))
	}
	if len(layers) > 0 {
		c.choiceCompositor = lipgloss.NewCompositor(layers...)
	} else {
		c.choiceCompositor = nil
	}
}

func (c *choiceList) HandleWheel(deltaX, deltaY float64) {
	if deltaY == 0 {
		return
	}
	c.scrollOffset += int(deltaY)
	c.wheelActive = true
	c.clampToBounds(c.lastLines, c.lastViewport)
}

func (c *choiceList) clampToBounds(lines []contentLine, viewport int) {
	limit := max(0, len(lines)-viewport)
	c.scrollOffset = min(max(0, c.scrollOffset), limit)
	if c.isFillIn() && c.fillInTop >= 0 {
		fillInBottom := c.fillInBottom
		if fillInBottom < 0 {
			fillInBottom = c.fillInTop
		}
		if c.scrollOffset > fillInBottom {
			c.scrollOffset = max(0, fillInBottom-viewport+1)
		}
		if c.scrollOffset+viewport <= c.fillInTop && fillInBottom-c.fillInTop+1 >= viewport {
			c.scrollOffset = c.fillInTop
		}
	}
}

func (c *choiceList) clampScroll(lines []contentLine, viewport int) {
	if c.suppressScroll {
		c.suppressScroll = false
		return
	}
	if c.wheelActive {
		c.clampToBounds(lines, viewport)
		return
	}
	limit := max(0, len(lines)-viewport)
	if limit == 0 {
		c.scrollOffset = 0
		return
	}

	cursorTop, cursorBottom := -1, -1
	for i, ln := range lines {
		if ln.cursorItem {
			if cursorTop < 0 {
				cursorTop = i
			}
			cursorBottom = i
		}
	}
	if c.isFillIn() && c.fillIn.Focused() && cursorTop >= 0 {
		if tc := c.fillIn.Cursor(); tc != nil {
			targetLine := cursorTop + tc.Y
			if targetLine >= cursorTop && targetLine <= cursorBottom {
				cursorTop = targetLine
				cursorBottom = targetLine
			}
		}
	}
	if cursorTop < 0 {
		c.scrollOffset = min(max(0, c.scrollOffset), limit)
		return
	}

	below := min(cursorBottom+1, len(lines)-1)
	if c.cursorIdx == 0 {
		c.scrollOffset = 0
	}
	if below >= c.scrollOffset+viewport {
		c.scrollOffset = below - viewport + 1
	}
	if cursorTop < c.scrollOffset {
		c.scrollOffset = cursorTop
	}
	c.scrollOffset = min(max(0, c.scrollOffset), limit)
}

func (c *choiceList) handleFillInFocused(
	msg tea.KeyPressMsg,
	doneKey key.Binding,
	onClose func() (bool, tea.Cmd),
	onDone func() (bool, tea.Cmd),
) (bool, tea.Cmd, bool) {
	if !c.isFillIn() || !c.fillIn.Focused() {
		return false, nil, false
	}
	if key.Matches(msg, c.keyClose) {
		done, cmd := onClose()
		return done, cmd, true
	}
	if key.Matches(msg, doneKey) {
		done, cmd := onDone()
		return done, cmd, true
	}
	cmd, handled := c.handleFillInKey(msg)
	return false, cmd, handled
}
