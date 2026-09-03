package model

import (
	"testing"

	"github.com/GiiS-AI/GiiS-Code/internal/question"
	"github.com/GiiS-AI/GiiS-Code/internal/ui/dialog"
	"github.com/stretchr/testify/require"
)

func newTestUIForQuestions() *UI {
	u := newTestUI()
	u.dialog = dialog.NewOverlay()
	return u
}

func TestHandleQuestionNotification_ClosesMatchingDialog(t *testing.T) {
	t.Parallel()

	u := newTestUIForQuestions()
	req := question.Request{
		ID: "batch-1",
		Questions: []question.Question{{
			ID:          "q-1",
			Type:        question.TypeYesNo,
			Text:        "Proceed?",
			Description: "Pick yes or no.",
		}},
	}
	u.dialog.OpenDialogWithGrace(dialog.NewQuestion(u.com, req))
	require.True(t, u.dialog.ContainsDialog(dialog.QuestionDialogID("batch-1")))

	u.handleQuestionNotification(question.Notification{BatchID: "batch-1"})

	require.False(t, u.dialog.ContainsDialog(dialog.QuestionDialogID("batch-1")))
}

func TestHandleQuestionNotification_IgnoresOtherBatches(t *testing.T) {
	t.Parallel()

	u := newTestUIForQuestions()
	req := question.Request{
		ID: "batch-keep",
		Questions: []question.Question{{
			ID:          "q-keep",
			Type:        question.TypeFreeText,
			Text:        "Why?",
			Description: "Explain.",
		}},
	}
	u.dialog.OpenDialogWithGrace(dialog.NewQuestion(u.com, req))

	u.handleQuestionNotification(question.Notification{BatchID: "batch-other"})

	require.True(t, u.dialog.ContainsDialog(dialog.QuestionDialogID("batch-keep")))
}
