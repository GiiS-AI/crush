package dialog

import (
	"image"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/question"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/common"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type FreeText struct {
	Styles  *styles.Styles
	Request question.Question
	focused bool

	editor       textarea.Model
	scrollOffset int
	wheelActive  bool
	keyEnter     key.Binding
	keyNewline   key.Binding
	keyClose     key.Binding

	lastResponse question.Answer
	lastWidth    int
}

const (
	freeTextMinEditorHeight = 3
	freeTextMaxEditorHeight = 6
)

func NewFreeText(sty *styles.Styles, req question.Question) *FreeText {
	ta := newQuestionTextarea(sty, "Type your answer...", 1000)
	ta.DynamicHeight = false
	ta.MinHeight = freeTextMinEditorHeight
	ta.MaxHeight = freeTextMaxEditorHeight
	ta.SetHeight(freeTextMinEditorHeight)

	return &FreeText{
		Styles:     sty,
		Request:    req,
		editor:     ta,
		keyEnter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		keyNewline: key.NewBinding(key.WithKeys("shift+enter", "ctrl+j"), key.WithHelp("shift+enter", "newline")),
		keyClose:   CloseKey,
	}
}

func (d *FreeText) HandleKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	d.wheelActive = false
	switch {
	case key.Matches(msg, d.keyClose):
		d.answer(question.Answer{QuestionID: d.Request.ID})
		return true, nil
	case key.Matches(msg, d.keyEnter):
		val := strings.TrimSpace(d.editor.Value())
		if val != "" {
			d.answer(question.Answer{
				QuestionID: d.Request.ID,
				FillInText: val,
			})
			return true, nil
		}
		return false, nil
	case key.Matches(msg, d.keyNewline):
		d.editor.InsertRune('\n')
		return false, nil
	default:
		var cmd tea.Cmd
		d.editor, cmd = d.editor.Update(msg)
		return false, cmd
	}
}

func (d *FreeText) answer(resp question.Answer) {
	d.lastResponse = resp
}

func (d *FreeText) Response() question.Answer {
	if val := strings.TrimSpace(d.editor.Value()); val != "" {
		return question.Answer{QuestionID: d.Request.ID, FillInText: val}
	}
	return d.lastResponse
}

func (d *FreeText) GetRequest() question.Question { return d.Request }

func (d *FreeText) ShortHelp() []key.Binding {
	return []key.Binding{d.keyEnter, d.keyNewline, d.keyClose}
}

func (d *FreeText) Height(width int) int {
	w := width
	if w <= 0 {
		w = d.lastWidth
	}
	if w <= 0 {
		w = choiceListMaxWidth
	}
	iconPrompt := questionIconPrompt(d.Styles, d.focused)
	h := sectionHeight(d.Request.Text, w-lipgloss.Width(iconPrompt))
	h++
	if d.Request.Description != "" {
		r := common.MarkdownRenderer(d.Styles, w)
		mu := common.LockMarkdownRenderer(r)
		mu.Lock()
		out, err := r.Render(d.Request.Description)
		mu.Unlock()
		if err == nil {
			out = strings.TrimSuffix(out, "\n")
			h += strings.Count(out, "\n") + 1
		} else {
			h += sectionHeight(d.Request.Description, w)
		}
		h++
	}
	h += freeTextMinEditorHeight
	h++
	return h
}

func (d *FreeText) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	d.lastWidth = area.Dx()
	viewport := area.Dy()

	barActive := d.Styles.Editor.QuestionCursorBar.Render("┃ ")
	const barInactive = "  "
	bar := barInactive
	if d.focused {
		bar = barActive
	}
	prefixWidth := lipgloss.Width(bar)
	iconPrompt := questionIconPrompt(d.Styles, d.focused)
	iconWidth := lipgloss.Width(iconPrompt)

	type ftLine struct {
		text    string
		cursorX int
	}

	build := func(contentWidth int) ([]ftLine, int) {
		var lines []ftLine
		cursorRow := -1

		header := iconPrompt + d.Styles.Editor.QuestionUnselected.Render(
			ansi.Wrap(d.Request.Text, contentWidth-iconWidth, ""),
		)
		for _, l := range strings.Split(header, "\n") {
			lines = append(lines, ftLine{text: l, cursorX: -1})
		}
		lines = append(lines, ftLine{cursorX: -1})

		if d.Request.Description != "" {
			r := common.MarkdownRenderer(d.Styles, contentWidth)
			mu := common.LockMarkdownRenderer(r)
			mu.Lock()
			desc, err := r.Render(d.Request.Description)
			mu.Unlock()
			if err != nil {
				desc = d.Request.Description
			}
			desc = strings.TrimSuffix(desc, "\n")
			for _, l := range strings.Split(desc, "\n") {
				lines = append(lines, ftLine{text: l, cursorX: -1})
			}
			lines = append(lines, ftLine{cursorX: -1})
		}

		headerLines := len(lines)
		fill := viewport - headerLines - 1
		available := min(freeTextMaxEditorHeight, max(freeTextMinEditorHeight, fill))
		d.editor.SetHeight(available)
		d.editor.SetWidth(contentWidth - 2 - prefixWidth)
		tc := d.editor.Cursor()
		for j, ln := range strings.Split(d.editor.View(), "\n") {
			text := bar + ln
			cursorX := -1
			if tc != nil && tc.Y == j {
				cursorRow = len(lines)
				cursorX = tc.X + prefixWidth
			}
			lines = append(lines, ftLine{text: text, cursorX: cursorX})
		}
		lines = append(lines, ftLine{cursorX: -1})
		return lines, cursorRow
	}

	contentWidth := area.Dx()
	lines, cursorRow := build(contentWidth)
	overflow := viewport > 0 && len(lines) > viewport
	if overflow {
		contentWidth--
		lines, cursorRow = build(contentWidth)
	}

	maxScroll := max(0, len(lines)-viewport)
	d.scrollOffset = min(max(0, d.scrollOffset), maxScroll)
	if !d.wheelActive && cursorRow >= 0 {
		if cursorRow < d.scrollOffset {
			d.scrollOffset = cursorRow
		} else if cursorRow >= d.scrollOffset+viewport {
			d.scrollOffset = cursorRow - viewport + 1
		}
		d.scrollOffset = min(max(0, d.scrollOffset), maxScroll)
	}

	var cur *tea.Cursor
	baseCursor := d.editor.Cursor()
	for screenRow := range viewport {
		idx := d.scrollOffset + screenRow
		if idx >= len(lines) {
			break
		}
		ln := lines[idx]
		y := area.Min.Y + screenRow
		drawStyledText(scr, image.Rect(area.Min.X, y, area.Min.X+contentWidth, y+1), ln.text)
		if ln.cursorX >= 0 && ln.cursorX < contentWidth && baseCursor != nil {
			c := *baseCursor
			c.X = ln.cursorX
			c.Y = screenRow
			cur = &c
		}
	}

	if overflow {
		sb := common.Scrollbar(d.Styles, viewport, len(lines), viewport, d.scrollOffset)
		if sb != "" {
			x := area.Max.X - 1
			uv.NewStyledString(sb).Draw(scr, image.Rect(x, area.Min.Y, x+1, area.Min.Y+viewport))
		}
	}

	return cur
}

func (d *FreeText) HeightChanged() bool { return false }
func (d *FreeText) SetFocused(focused bool) {
	d.focused = focused
	if focused {
		d.editor.Focus()
	} else {
		d.editor.Blur()
	}
}
func (d *FreeText) SetHover(x, y int)                      {}
func (d *FreeText) HandleMouseClick(x, y int) (bool, bool) { return false, false }
func (d *FreeText) HandleWheel(deltaX, deltaY float64) {
	if deltaY != 0 {
		d.scrollOffset += int(deltaY)
		d.wheelActive = true
	}
}
func (d *FreeText) HandlePaste(msg tea.PasteMsg) tea.Cmd {
	var cmd tea.Cmd
	d.editor, cmd = d.editor.Update(msg)
	return cmd
}
