package backend

import (
	"github.com/GiiS-AI/GiiS-Code/internal/proto"
	"github.com/GiiS-AI/GiiS-Code/internal/question"
)

// AnswerQuestion submits answers for a question batch.
func (b *Backend) AnswerQuestion(workspaceID string, req proto.QuestionAnswer) (bool, error) {
	ws, err := b.GetWorkspace(workspaceID)
	if err != nil {
		return false, err
	}

	responses := make([]question.Answer, len(req.Responses))
	for i, r := range req.Responses {
		responses[i] = question.Answer{
			QuestionID:  r.QuestionID,
			SelectedIDs: r.SelectedIDs,
			FillInText:  r.FillInText,
			Yes:         r.Yes,
			Notes:       r.Notes,
		}
	}

	return ws.Questions.Answer(req.BatchRequestID, responses), nil
}

// CancelQuestion cancels a pending question batch.
func (b *Backend) CancelQuestion(workspaceID string, req proto.QuestionCancel) (bool, error) {
	ws, err := b.GetWorkspace(workspaceID)
	if err != nil {
		return false, err
	}
	return ws.Questions.Cancel(req.BatchRequestID), nil
}
