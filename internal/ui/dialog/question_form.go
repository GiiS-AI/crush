package dialog

import (
	"fmt"
	"image"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/question"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type questionResponder interface {
	HandleKey(msg tea.KeyPressMsg) (bool, tea.Cmd)
	Response() question.Answer
	SetHover(x, y int)
	HandleMouseClick(x, y int) (done bool, handled bool)
	HandlePaste(msg tea.PasteMsg) tea.Cmd
	Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor
	Height(width int) int
	HeightChanged() bool
	SetFocused(focused bool)
	ShortHelp() []key.Binding
	GetRequest() question.Question
}

type wheelScrollableQuestion interface {
	HandleWheel(deltaX, deltaY float64)
}

type QuestionForm struct {
	Styles       *styles.Styles
	BatchID      string
	questions    []questionResponder
	labels       []string
	requestIDs   []string
	answers      []*question.Answer
	activeIdx    int
	focused      bool
	hasConfirm   bool
	showTabs     bool
	numQuestions int
	confirmComp  *ConfirmComponent

	keyPrevTab key.Binding
	keyNextTab key.Binding
	keyClose   key.Binding

	compositor *lipgloss.Compositor
	hoverX     int
	hoverY     int

	submitted bool
	cancelled bool
}

func NewQuestionForm(sty *styles.Styles, batch question.Request) *QuestionForm {
	comps := make([]questionResponder, len(batch.Questions))
	labels := make([]string, len(batch.Questions))
	ids := make([]string, len(batch.Questions))
	for i, req := range batch.Questions {
		switch req.Type {
		case question.TypeYesNo:
			comps[i] = NewYesNo(sty, req)
		case question.TypeSingleChoice:
			comps[i] = NewSingleChoice(sty, req)
		case question.TypeMultiChoice:
			comps[i] = NewMultiChoice(sty, req)
		case question.TypeFreeText:
			comps[i] = NewFreeText(sty, req)
		}
		if req.Label != "" {
			labels[i] = req.Label
		} else {
			labels[i] = shortLabel(req.Text)
		}
		ids[i] = req.ID
	}

	numQuestions := len(comps)
	hasConfirm := numQuestions > 1
	answers := make([]*question.Answer, numQuestions)

	var confirmComp *ConfirmComponent
	allLabels := labels
	if hasConfirm {
		confirmTitle := batch.ConfirmTitle
		if confirmTitle == "" {
			confirmTitle = "Confirm"
		}
		confirmComp = NewConfirmComponent(
			sty,
			confirmTitle,
			batch.ConfirmDescription,
			labels,
			batch.Questions,
			answers,
		)
		allLabels = make([]string, len(labels)+1)
		copy(allLabels, labels)
		allLabels[len(labels)] = "Confirm"
	}
	showTabs := numQuestions > 1

	f := &QuestionForm{
		Styles:       sty,
		BatchID:      batch.ID,
		questions:    comps,
		labels:       allLabels,
		requestIDs:   ids,
		answers:      answers,
		hasConfirm:   hasConfirm,
		showTabs:     showTabs,
		numQuestions: numQuestions,
		confirmComp:  confirmComp,
		keyPrevTab:   key.NewBinding(key.WithKeys("[", "ctrl+left"), key.WithHelp("[", "prev tab")),
		keyNextTab:   key.NewBinding(key.WithKeys("]", "ctrl+right"), key.WithHelp("]", "next tab")),
		keyClose:     CloseKey,
	}

	if confirmComp != nil {
		confirmComp.OnConfirm = f.submit
		confirmComp.OnReject = func() {
			if idx := f.firstUnanswered(); idx >= 0 {
				f.switchTab(idx)
			} else if numQuestions > 0 {
				f.switchTab(numQuestions - 1)
			}
		}
	}

	if len(comps) > 0 {
		comps[0].SetFocused(true)
	}
	return f
}

func shortLabel(q string) string {
	q = strings.ReplaceAll(q, "\n", " ")
	words := strings.Fields(q)
	if len(words) > 3 {
		words = words[:3]
	}
	return strings.Join(words, " ")
}

func (f *QuestionForm) isConfirmTab() bool {
	return f.hasConfirm && f.activeIdx == f.numQuestions
}

func (f *QuestionForm) isAnswered(idx int) bool {
	if idx >= len(f.answers) || f.answers[idx] == nil {
		return false
	}
	resp := f.answers[idx]
	return len(resp.SelectedIDs) > 0 || resp.FillInText != "" || resp.Yes != nil
}

func (f *QuestionForm) firstUnanswered() int {
	for i, ans := range f.answers {
		if ans == nil {
			return i
		}
		if len(ans.SelectedIDs) == 0 && ans.FillInText == "" && ans.Yes == nil {
			return i
		}
	}
	return -1
}

func (f *QuestionForm) HandleKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, f.keyNextTab):
		f.switchTab(f.activeIdx + 1)
		return false, nil
	case key.Matches(msg, f.keyPrevTab):
		f.switchTab(f.activeIdx - 1)
		return false, nil
	}

	if f.isConfirmTab() {
		done, cmd := f.confirmComp.HandleKey(msg)
		if done {
			return true, cmd
		}
		return false, cmd
	}

	if key.Matches(msg, f.keyClose) {
		f.cancel()
		return true, nil
	}

	if f.activeIdx < f.numQuestions {
		done, cmd := f.questions[f.activeIdx].HandleKey(msg)
		if done {
			resp := f.questions[f.activeIdx].Response()
			f.answers[f.activeIdx] = &resp
			f.syncConfirmAnswers()
			if f.activeIdx < len(f.labels)-1 {
				f.switchTab(f.activeIdx + 1)
			} else if !f.hasConfirm {
				f.submit()
				return true, cmd
			}
			return false, cmd
		}
		return false, cmd
	}
	return false, nil
}

func (f *QuestionForm) HandleWheel(deltaX, deltaY float64) {
	if f.isConfirmTab() {
		if deltaY < 0 && f.confirmComp.scrollOffset > 0 {
			f.confirmComp.scrollOffset--
		} else if deltaY > 0 {
			f.confirmComp.scrollOffset++
		}
		return
	}
	if f.activeIdx >= f.numQuestions {
		return
	}
	if we, ok := f.questions[f.activeIdx].(wheelScrollableQuestion); ok {
		we.HandleWheel(deltaX, deltaY)
	}
}

func (f *QuestionForm) switchTab(idx int) {
	totalTabs := len(f.labels)
	if totalTabs == 0 {
		return
	}
	if !f.isConfirmTab() && f.activeIdx < f.numQuestions {
		resp := f.questions[f.activeIdx].Response()
		f.answers[f.activeIdx] = &resp
		f.questions[f.activeIdx].SetFocused(false)
	} else if f.isConfirmTab() {
		f.confirmComp.SetFocused(false)
	}
	if idx < 0 {
		idx = totalTabs - 1
	} else if idx >= totalTabs {
		idx = 0
	}
	f.activeIdx = idx
	if f.isConfirmTab() {
		f.syncConfirmAnswers()
		f.confirmComp.SetFocused(f.focused)
	} else if f.activeIdx < f.numQuestions {
		f.questions[f.activeIdx].SetFocused(f.focused)
	}
}

func (f *QuestionForm) syncConfirmAnswers() {
	if f.confirmComp != nil {
		f.confirmComp.UpdateAnswers(f.answers)
	}
}

func (f *QuestionForm) submit() {
	f.submitted = true
	f.cancelled = false
}

func (f *QuestionForm) cancel() {
	f.cancelled = true
	f.submitted = false
}

func (f *QuestionForm) Answers() []question.Answer {
	responses := make([]question.Answer, f.numQuestions)
	for i, ans := range f.answers {
		if ans != nil {
			responses[i] = *ans
		} else {
			responses[i] = question.Answer{QuestionID: f.requestIDs[i]}
		}
	}
	return responses
}

func (f *QuestionForm) Submitted() bool { return f.submitted }
func (f *QuestionForm) Cancelled() bool { return f.cancelled }

func (f *QuestionForm) ShortHelp() []key.Binding {
	if f.isConfirmTab() {
		return f.confirmComp.ShortHelp()
	}
	bindings := []key.Binding{f.keyPrevTab, f.keyNextTab}
	if f.activeIdx < f.numQuestions {
		bindings = append(bindings, f.questions[f.activeIdx].ShortHelp()...)
	}
	return bindings
}

func (f *QuestionForm) Height(width int) int {
	h := 0
	if f.showTabs {
		h = 4
	}
	maxQ := 0
	for _, q := range f.questions {
		if qh := q.Height(width); qh > maxQ {
			maxQ = qh
		}
	}
	if f.confirmComp != nil {
		if ch := f.confirmComp.Height(width); ch > maxQ {
			maxQ = ch
		}
	}
	h += maxQ
	return h
}

func (f *QuestionForm) DrawCollapsed(scr uv.Screen, area uv.Rectangle) {
	icon := f.Styles.Editor.PromptQuestionIconBlurred.Render()
	iconWidth := lipgloss.Width(icon)
	textStyle := f.Styles.Messages.AssistantInfoModel
	countStyle := f.Styles.Messages.AssistantInfoProvider
	lineStyle := f.Styles.Section.Line

	var plainText string
	var confirmRendered string
	if f.numQuestions > 1 {
		answered := 0
		for i := 0; i < f.numQuestions; i++ {
			if f.isAnswered(i) {
				answered++
			}
		}
		if f.isConfirmTab() && f.confirmComp != nil {
			plainText = f.confirmComp.Title
			confirmRendered = f.Styles.Editor.QuestionUnselected.Render(f.confirmComp.Title)
		} else if f.activeIdx < len(f.questions) {
			plainText = f.getQuestionText(f.activeIdx)
		}
		count := fmt.Sprintf("(%d/%d answered)", answered, f.numQuestions)
		plainLabel := plainText + " " + count
		textWidth := iconWidth + 1 + lipgloss.Width(plainLabel)
		remaining := area.Dx() - textWidth - 1

		var rendered string
		if confirmRendered != "" {
			rendered = fmt.Sprintf("%s%s %s", icon, confirmRendered, countStyle.Render(count))
		} else {
			rendered = fmt.Sprintf("%s%s %s", icon, textStyle.Render(plainText), countStyle.Render(count))
		}
		if remaining > 0 {
			rendered = rendered + " " + lineStyle.Render(strings.Repeat(styles.SectionSeparator, remaining))
		}
		drawStyledText(scr, area, rendered)
	} else if f.numQuestions == 1 {
		plainText = f.getQuestionText(0)
		textWidth := iconWidth + 1 + lipgloss.Width(plainText)
		remaining := area.Dx() - textWidth - 1
		rendered := fmt.Sprintf("%s%s", icon, textStyle.Render(plainText))
		if remaining > 0 {
			rendered = rendered + " " + lineStyle.Render(strings.Repeat(styles.SectionSeparator, remaining))
		}
		drawStyledText(scr, area, rendered)
	}
}

func (f *QuestionForm) getQuestionText(idx int) string {
	if idx < len(f.questions) {
		return f.questions[idx].GetRequest().Text
	}
	if idx < len(f.labels) {
		return f.labels[idx]
	}
	return ""
}

func (f *QuestionForm) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	contentY := area.Min.Y

	if f.showTabs {
		const tabPadX = 1
		tabHeight := 3

		labels := make([]string, len(f.labels))
		copy(labels, f.labels)

		tabWidths := make([]int, len(labels))
		naturalWidths := make([]int, len(labels))
		totalWidth := 0
		for i, l := range labels {
			w := ansi.StringWidth(l) + tabPadX*2 + 2
			tabWidths[i] = w
			naturalWidths[i] = w
			totalWidth += w
		}
		avail := area.Dx()
		if totalWidth > avail && len(labels) > 0 {
			const minLabelW = 1
			minTabW := minLabelW + tabPadX*2 + 2
			n := len(labels)

			usefulMinTabW := 5 + tabPadX*2 + 2
			if avail/n < usefulMinTabW {
				counter := fmt.Sprintf("%d/%d", f.activeIdx+1, n)
				activeLabel := labels[f.activeIdx]
				combined := activeLabel + " · " + counter
				maxLabel := avail - tabPadX*2 - 2
				if maxLabel < 3 {
					maxLabel = 3
				}
				if ansi.StringWidth(combined) > maxLabel {
					counterPart := " · " + counter
					labelBudget := maxLabel - ansi.StringWidth(counterPart)
					if labelBudget < 1 {
						labelBudget = 1
					}
					combined = ansi.Truncate(activeLabel, labelBudget, "…") + counterPart
				}
				for i := range labels {
					if i == f.activeIdx {
						labels[i] = combined
					} else {
						labels[i] = ""
					}
				}
				totalWidth = 0
				for i := range labels {
					if labels[i] == "" {
						tabWidths[i] = 0
					} else {
						w := ansi.StringWidth(labels[i]) + tabPadX*2 + 2
						tabWidths[i] = w
						totalWidth += w
					}
				}
			} else {
				capped := make([]bool, n)
				for {
					freeCount := 0
					freeTotal := 0
					for i := range n {
						if capped[i] {
							continue
						}
						freeCount++
						freeTotal += naturalWidths[i]
					}
					if freeCount == 0 {
						break
					}
					budget := avail
					for i := range n {
						if capped[i] {
							budget -= tabWidths[i]
						}
					}
					share := budget / freeCount
					changed := false
					for i := range n {
						if !capped[i] && naturalWidths[i] <= share {
							capped[i] = true
							tabWidths[i] = naturalWidths[i]
							changed = true
						}
					}
					if !changed {
						for i := range n {
							if !capped[i] {
								tabWidths[i] = max(share, minTabW)
							}
						}
						remainder := budget - share*freeCount
						for i := range n {
							if remainder <= 0 {
								break
							}
							if !capped[i] && tabWidths[i] < naturalWidths[i] {
								tabWidths[i]++
								remainder--
							}
						}
						break
					}
				}

				for i, l := range labels {
					labelAvail := max(tabWidths[i]-tabPadX*2-2, minLabelW)
					if ansi.StringWidth(l) > labelAvail {
						labels[i] = ansi.Truncate(l, labelAvail, "…")
					}
				}
			}
		}

		var layers []*lipgloss.Layer
		x := area.Min.X

		hoveredTab := -1
		if f.hoverY >= area.Min.Y && f.hoverY < area.Min.Y+tabHeight {
			tx := area.Min.X
			for i := range labels {
				tw := tabWidths[i]
				if f.hoverX >= tx && f.hoverX < tx+tw {
					hoveredTab = i
					break
				}
				tx += tw
			}
		}

		firstVisible := -1
		for i := range labels {
			if tabWidths[i] > 0 {
				firstVisible = i
				break
			}
		}

		for i, label := range labels {
			if tabWidths[i] == 0 {
				continue
			}
			isActive := i == f.activeIdx
			isHovered := i == hoveredTab && !isActive
			labelWidth := ansi.StringWidth(label)
			tabWidth := tabWidths[i]

			tabArea := image.Rect(x, area.Min.Y, x+tabWidth, area.Min.Y+tabHeight)

			border := f.Styles.Tab.InactiveBorder
			textStyle := f.Styles.Tab.InactiveStyle
			if !f.focused {
				border = f.Styles.Tab.InactiveBorderBlurred
			}
			if isActive {
				border = f.Styles.Tab.ActiveBorder
				textStyle = f.Styles.Tab.ActiveStyle
				if !f.focused {
					border = f.Styles.Tab.ActiveBorderBlurred
				}
			} else if i < f.numQuestions && f.isAnswered(i) {
				textStyle = f.Styles.Tab.ActiveStyle
			}
			if isHovered {
				hovered := textStyle
				hovered.Attrs |= uv.AttrBold
				textStyle = hovered
			}

			if i == firstVisible {
				if isActive {
					border.BottomLeft = uv.Side{Content: "┘", Style: border.BottomLeft.Style}
				} else {
					border.BottomLeft = uv.Side{Content: "┴", Style: border.BottomLeft.Style}
				}
			}

			border.Draw(scr, tabArea)

			innerWidth := tabWidth - 2
			xOff := (innerWidth - labelWidth) / 2
			innerArea := image.Rect(tabArea.Min.X+1+xOff, tabArea.Min.Y+1, tabArea.Max.X-1, tabArea.Max.Y-1)
			uv.NewStyledString(textStyle.Styled(label)).Draw(scr, innerArea)

			hitStr := strings.Repeat(strings.Repeat(" ", tabWidth)+"\n", tabHeight-1) + strings.Repeat(" ", tabWidth)
			layers = append(layers, lipgloss.NewLayer(hitStr).X(x).Y(area.Min.Y).ID(fmt.Sprintf("tab_%d", i)))

			x += tabWidth
		}

		f.compositor = lipgloss.NewCompositor(layers...)

		lineY := area.Min.Y + tabHeight - 1
		lineSide := f.Styles.Tab.InactiveBorder.Bottom
		if !f.focused {
			lineSide = f.Styles.Tab.InactiveBorderBlurred.Bottom
		}
		for lx := x; lx < area.Max.X; lx++ {
			c := uv.NewCell(scr.WidthMethod(), lineSide.Content)
			if c != nil {
				c.Style = lineSide.Style
			}
			scr.SetCell(lx, lineY, c)
		}

		contentY = area.Min.Y + tabHeight + 1
	} else {
		f.compositor = nil
	}

	contentArea := image.Rect(area.Min.X, contentY, area.Max.X, area.Max.Y)

	if f.isConfirmTab() {
		return f.confirmComp.Draw(scr, contentArea)
	}
	if f.activeIdx < f.numQuestions {
		cur := f.questions[f.activeIdx].Draw(scr, contentArea)
		if cur != nil {
			cur.Y += contentY - area.Min.Y
		}
		return cur
	}
	return nil
}

func (f *QuestionForm) HeightChanged() bool {
	for _, q := range f.questions {
		if q.HeightChanged() {
			return true
		}
	}
	if f.confirmComp != nil && f.confirmComp.HeightChanged() {
		return true
	}
	return false
}

func (f *QuestionForm) SetFocused(focused bool) {
	f.focused = focused
	if f.isConfirmTab() {
		f.confirmComp.SetFocused(focused)
	} else if f.activeIdx < f.numQuestions {
		f.questions[f.activeIdx].SetFocused(focused)
	}
}

func (f *QuestionForm) SetHover(x, y int) {
	f.hoverX = x
	f.hoverY = y
	if f.isConfirmTab() && f.confirmComp != nil {
		f.confirmComp.SetHover(x, y)
	} else if f.activeIdx < len(f.questions) {
		f.questions[f.activeIdx].SetHover(x, y)
	}
}

func (f *QuestionForm) HandlePaste(msg tea.PasteMsg) tea.Cmd {
	if f.isConfirmTab() {
		return nil
	}
	if f.activeIdx < f.numQuestions {
		return f.questions[f.activeIdx].HandlePaste(msg)
	}
	return nil
}

func (f *QuestionForm) HandleMouseClick(x, y int) (bool, bool) {
	if f.showTabs && f.compositor != nil {
		hit := f.compositor.Hit(x, y)
		if !hit.Empty() {
			var idx int
			if _, err := fmt.Sscanf(hit.ID(), "tab_%d", &idx); err == nil {
				if idx >= 0 && idx < len(f.labels) && idx != f.activeIdx {
					f.switchTab(idx)
				}
				return false, true
			}
		}
	}

	if f.isConfirmTab() && f.confirmComp != nil {
		return f.confirmComp.HandleMouseClick(x, y)
	}
	if f.activeIdx < f.numQuestions {
		return f.questions[f.activeIdx].HandleMouseClick(x, y)
	}
	return false, false
}
