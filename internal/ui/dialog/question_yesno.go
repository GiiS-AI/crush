package dialog

import (
	"image"
	"maps"
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

type YesNo struct {
	questionEditor
	Request    question.Question
	selectedNo bool
	focused    bool
	compositor *lipgloss.Compositor
	hoverX     int
	hoverY     int

	keyLeftRight key.Binding
	keyEnter     key.Binding
	keyYes       key.Binding
	keyNo        key.Binding
	keyClose     key.Binding

	lastResponse question.Answer
	lastWidth    int
}

func NewYesNo(sty *styles.Styles, req question.Question) *YesNo {
	return &YesNo{
		questionEditor: newQuestionEditor(sty),
		Request:        req,
		selectedNo:     true,
		keyLeftRight:   key.NewBinding(key.WithKeys("left", "right", "h", "l"), key.WithHelp("←/→", "switch")),
		keyEnter:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		keyYes:         key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "yes")),
		keyNo:          key.NewBinding(key.WithKeys("n", "N"), key.WithHelp("n", "no")),
		keyClose:       CloseKey,
	}
}

func (d *YesNo) HandleKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if d.activeNoteKey != "" && d.noteEditor.Focused() {
		cmd, handled := d.handleNoteKey(msg, d.keyClose, func() { d.closeNote("_question") })
		if handled {
			return false, cmd
		}
	}

	switch {
	case key.Matches(msg, CloseKey):
		d.answer(question.Answer{QuestionID: d.Request.ID})
		return true, nil
	case key.Matches(msg, d.keyLeftRight):
		d.selectedNo = !d.selectedNo
		return false, nil
	case key.Matches(msg, d.keyEnter):
		d.answer(d.respond(!d.selectedNo))
		return true, nil
	case key.Matches(msg, d.keyYes):
		d.answer(d.respond(true))
		return true, nil
	case key.Matches(msg, d.keyNo):
		d.answer(d.respond(false))
		return true, nil
	case key.Matches(msg, d.keyNote):
		return false, d.openNote("_question")
	}
	return false, nil
}

func (d *YesNo) answer(resp question.Answer) {
	d.lastResponse = resp
}

func (d *YesNo) Response() question.Answer     { return d.respond(!d.selectedNo) }
func (d *YesNo) GetRequest() question.Question { return d.Request }

func (d *YesNo) ShortHelp() []key.Binding {
	if d.activeNoteKey != "" && d.noteEditor.Focused() {
		return []key.Binding{d.keyClose, key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save note"))}
	}
	return []key.Binding{d.keyLeftRight, d.keyEnter, d.keyYes, d.keyNo, d.keyNote}
}

func (d *YesNo) respond(yes bool) question.Answer {
	resp := question.Answer{
		QuestionID: d.Request.ID,
		Yes:        &yes,
	}
	if len(d.notes) > 0 {
		resp.Notes = make(map[string]string, len(d.notes))
		maps.Copy(resp.Notes, d.notes)
	}
	return resp
}

func (d *YesNo) Height(width int) int {
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
	h++
	if d.activeNoteKey != "" && d.noteEditor.Focused() {
		h++
		h += d.noteEditor.Height()
	} else if len(d.notes) > 0 {
		h++
		h++
	}
	h++
	return h
}

func (d *YesNo) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	d.lastWidth = area.Dx()
	y := area.Min.Y

	iconPrompt := questionIconPrompt(d.Styles, d.focused)
	qText := iconPrompt + d.Styles.Editor.QuestionUnselected.Render(
		ansi.Wrap(d.Request.Text, area.Dx()-lipgloss.Width(iconPrompt), ""),
	)
	y += drawStyledText(scr, image.Rect(area.Min.X, y, area.Max.X, area.Max.Y), qText)
	y++

	if d.Request.Description != "" {
		r := common.MarkdownRenderer(d.Styles, area.Dx())
		mu := common.LockMarkdownRenderer(r)
		mu.Lock()
		desc, err := r.Render(d.Request.Description)
		mu.Unlock()
		if err == nil {
			desc = strings.TrimSuffix(desc, "\n")
			y += drawStyledText(scr, image.Rect(area.Min.X, y, area.Max.X, area.Max.Y), desc)
		} else {
			y += drawStyledText(scr, image.Rect(area.Min.X, y, area.Max.X, area.Max.Y), d.Request.Description)
		}
		y++
	}

	buttonOptsList := []common.ButtonOpts{
		{Text: "Yes", Selected: !d.selectedNo, Padding: 3, UnderlineIndex: 0},
		{Text: "No", Selected: d.selectedNo, Padding: 3, UnderlineIndex: 0},
	}
	d.compositor = common.ButtonHitCompositor(d.Styles, buttonOptsList, " ", area.Min.X, y)
	hoveredBtn := common.HitButtonIndex(d.compositor, d.hoverX, d.hoverY)
	buttonOptsList[0].Hovered = hoveredBtn == 0
	buttonOptsList[1].Hovered = hoveredBtn == 1
	buttons := common.ButtonGroup(d.Styles, buttonOptsList, " ")
	y += drawStyledText(scr, image.Rect(area.Min.X, y, area.Max.X, area.Max.Y), buttons)

	cur, _ := d.drawStandaloneNote(scr, area, y, "_question")
	return cur
}

func (d *YesNo) HeightChanged() bool                  { return false }
func (d *YesNo) SetFocused(focused bool)              { d.focused = focused }
func (d *YesNo) SetHover(x, y int)                    { d.hoverX = x; d.hoverY = y }
func (d *YesNo) HandlePaste(msg tea.PasteMsg) tea.Cmd { return d.handlePaste(msg) }

func (d *YesNo) HandleMouseClick(x, y int) (bool, bool) {
	switch common.HitButtonIndex(d.compositor, x, y) {
	case 0:
		d.selectedNo = false
		d.answer(d.respond(true))
		return true, true
	case 1:
		d.selectedNo = true
		d.answer(d.respond(false))
		return true, true
	}
	return false, false
}
