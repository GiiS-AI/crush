package proto

// QuestionRequest is the wire format for a question batch delivered over SSE.
type QuestionRequest struct {
	ID                 string         `json:"id"`
	SessionID          string         `json:"session_id"`
	ToolCallID         string         `json:"tool_call_id"`
	Questions          []QuestionItem `json:"questions"`
	ConfirmTitle       string         `json:"confirm_title,omitempty"`
	ConfirmDescription string         `json:"confirm_description,omitempty"`
}

// QuestionItem is a single question within a batch.
type QuestionItem struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Label       string           `json:"label,omitempty"`
	Question    string           `json:"question"`
	Description string           `json:"description,omitempty"`
	Choices     []QuestionChoice `json:"choices,omitempty"`
}

// QuestionChoice is a selectable option.
type QuestionChoice struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// QuestionAnswer carries a batch response from client to server.
type QuestionAnswer struct {
	BatchRequestID string             `json:"batch_request_id"`
	Responses      []QuestionResponse `json:"responses"`
}

// QuestionCancel identifies which pending question batch to cancel.
type QuestionCancel struct {
	BatchRequestID string `json:"batch_request_id"`
}

// QuestionResponse is a single answer within a batch response.
type QuestionResponse struct {
	QuestionID  string            `json:"question_id"`
	SelectedIDs []string          `json:"selected_ids,omitempty"`
	FillInText  string            `json:"fill_in_text,omitempty"`
	Yes         *bool             `json:"yes,omitempty"`
	Notes       map[string]string `json:"notes,omitempty"`
}

// QuestionAnswerResponse reports whether an answer or cancel resolved a batch.
type QuestionAnswerResponse struct {
	Resolved bool `json:"resolved"`
}

// QuestionNotification is published when a batch is resolved.
type QuestionNotification struct {
	BatchID string `json:"batch_id"`
}
