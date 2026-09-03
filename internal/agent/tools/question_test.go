package tools

import (
	"encoding/json"
	"testing"

	"github.com/GiiS-AI/GiiS-Code/internal/question"
	"github.com/stretchr/testify/require"
)

func TestQuestionParamsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	rawArray := []byte(`{
		"questions": [{"type":"yes_no","question":"Proceed?","description":"Pick yes or no."}]
	}`)
	var params QuestionParams
	require.NoError(t, json.Unmarshal(rawArray, &params))
	require.Len(t, params.Questions, 1)
	require.Equal(t, "yes_no", params.Questions[0].Type)

	rawString := []byte(`{
		"questions": "[{\"type\":\"free_text\",\"question\":\"Why?\",\"description\":\"Explain.\"}]"
	}`)
	require.NoError(t, json.Unmarshal(rawString, &params))
	require.Len(t, params.Questions, 1)
	require.Equal(t, "free_text", params.Questions[0].Type)
}

func TestNewQuestionToolInfo(t *testing.T) {
	t.Parallel()

	tool := NewQuestionTool(question.NewService())
	info := tool.Info()
	require.Equal(t, QuestionToolName, info.Name)
	require.NotEmpty(t, info.Description)
	require.Contains(t, info.Required, "questions")
	require.Contains(t, info.Parameters, "questions")
}

func TestFormatQuestionAnswer(t *testing.T) {
	t.Parallel()

	resp, err := formatQuestionAnswer(&question.Answer{
		QuestionID:  "q-1",
		SelectedIDs: []string{"alpha", "beta"},
		FillInText:  "other",
		Notes: map[string]string{
			"_question": "top-level note",
			"alpha":     "choice note",
		},
	})
	require.NoError(t, err)
	require.Contains(t, resp.Content, `User selected: ["alpha","beta"]`)
	require.Contains(t, resp.Content, "User provided: other")
	require.Contains(t, resp.Content, "Notes:")
	require.Contains(t, resp.Content, "- top-level note")
	require.Contains(t, resp.Content, "- [alpha]: choice note")
}
