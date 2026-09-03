package dialog

import (
	"image"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/question"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/common"
	uv "github.com/charmbracelet/ultraviolet"
)

const questionDialogIDPrefix = "question:"

// QuestionDialogID returns the overlay dialog ID for a question batch.
func QuestionDialogID(batchID string) string {
	return questionDialogIDPrefix + batchID
}

// Questions presents a blocking question batch inside a dialog overlay.
type Questions struct {
	com   *common.Common
	batch question.Request
	form  *QuestionForm
	help  help.Model
}

var _ Dialog = (*Questions)(nil)

// NewQuestion creates a new question dialog for a batch request.
func NewQuestion(com *common.Common, batch question.Request) *Questions {
	h := help.New()
	h.Styles = com.Styles.DialogHelpStyles()

	form := NewQuestionForm(com.Styles, batch)
	dlg := &Questions{
		com:   com,
		batch: batch,
		form:  form,
		help:  h,
	}
	return dlg
}

// ID implements Dialog.
func (q *Questions) ID() string {
	return QuestionDialogID(q.batch.ID)
}

// BatchID returns the underlying question batch ID.
func (q *Questions) BatchID() string {
	return q.batch.ID
}

// HandleMsg implements Dialog.
func (q *Questions) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		_, cmd := q.form.HandleKey(msg)
		if cmd != nil {
			return ActionCmd{Cmd: cmd}
		}
	case tea.MouseMotionMsg:
		q.form.SetHover(msg.X, msg.Y)
	case tea.MouseClickMsg:
		q.form.HandleMouseClick(msg.X, msg.Y)
	case common.CoalescedWheelMsg:
		q.form.HandleWheel(msg.DeltaX, msg.DeltaY)
	case tea.PasteMsg:
		if cmd := q.form.HandlePaste(msg); cmd != nil {
			return ActionCmd{Cmd: cmd}
		}
	}

	if q.form.Cancelled() {
		return ActionQuestionCancel{BatchID: q.batch.ID}
	}
	if q.form.Submitted() {
		return ActionQuestionResponse{
			BatchID:   q.batch.ID,
			Responses: q.form.Answers(),
		}
	}
	return nil
}

// Draw implements Dialog.
func (q *Questions) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := q.com.Styles
	width := min(max(area.Dx()*3/4, 70), 110)
	width = min(width, area.Dx())

	title := "Question"
	if len(q.batch.Questions) > 1 {
		title = "Questions"
	}

	helpView := q.help.View(q)
	rc := NewRenderContext(t, width)
	rc.Title = title
	rc.Gap = 1
	rc.Help = helpView

	contentWidth := width - rc.ViewStyle.GetHorizontalFrameSize() - 2
	formHeight := max(8, q.form.Height(contentWidth))
	rc.AddPart(strings.Repeat("\n", max(formHeight-1, 0)))

	headerHeight := 0
	if rc.Title != "" {
		header := common.DialogTitle(t, rc.Title, max(0, width-rc.ViewStyle.GetHorizontalFrameSize()-t.Dialog.Title.GetHorizontalFrameSize()), rc.TitleGradientFromColor, rc.TitleGradientToColor)
		headerHeight = lipgloss.Height(t.Dialog.Title.Render(header)) + rc.Gap
	}
	view := rc.Render()
	dialogHeight := min(lipgloss.Height(view), area.Dy())
	center := common.CenterRect(area, width, dialogHeight)
	DrawCenterCursor(scr, center, view, nil)

	frame := rc.ViewStyle.Width(width).Padding(0, 1)
	helpHeight := 0
	if helpView != "" {
		helpHeight = lipgloss.Height(helpView) + rc.Gap
	}
	contentArea := image.Rect(
		center.Min.X+frame.GetBorderLeftSize()+frame.GetPaddingLeft()+1,
		center.Min.Y+frame.GetBorderTopSize()+frame.GetPaddingTop()+headerHeight,
		center.Max.X-frame.GetBorderRightSize()-frame.GetPaddingRight()-1,
		center.Max.Y-frame.GetBorderBottomSize()-frame.GetPaddingBottom()-helpHeight,
	)
	cur := q.form.Draw(scr, contentArea)
	return cur
}

func (q *Questions) ShortHelp() []key.Binding {
	return q.form.ShortHelp()
}

func (q *Questions) FullHelp() [][]key.Binding {
	return [][]key.Binding{q.ShortHelp()}
}
