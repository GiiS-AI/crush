// Package question provides services for asking the user structured
// questions via the UI and blocking until an answer is received.
package question

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/GiiS-AI/GiiS-Code/internal/csync"
	"github.com/GiiS-AI/GiiS-Code/internal/pubsub"
	"github.com/google/uuid"
)

// ErrCancelled is returned by Ask when the user cancels the question.
var ErrCancelled = errors.New("question cancelled by user")

// Type identifies the kind of question to present.
type Type string

const (
	TypeYesNo        Type = "yes_no"
	TypeSingleChoice Type = "single_choice"
	TypeMultiChoice  Type = "multi_choice"
	TypeFreeText     Type = "free_text"
)

// Choice represents a single selectable option.
type Choice struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Question is a single question definition within a Request.
type Question struct {
	ID          string   `json:"id"`
	Type        Type     `json:"type"`
	Label       string   `json:"label,omitempty"`
	Text        string   `json:"question"`
	Description string   `json:"description,omitempty"`
	Choices     []Choice `json:"choices,omitempty"`
}

// Answer carries the user's response to a single Question.
type Answer struct {
	QuestionID  string            `json:"question_id"`
	SelectedIDs []string          `json:"selected_ids,omitempty"`
	FillInText  string            `json:"fill_in_text,omitempty"`
	Yes         *bool             `json:"yes,omitempty"`
	Notes       map[string]string `json:"notes,omitempty"`
}

// HasNotes reports whether any notes were attached.
func (a Answer) HasNotes() bool { return len(a.Notes) > 0 }

// Request is the service envelope published to the UI.
type Request struct {
	ID                 string     `json:"id"`
	SessionID          string     `json:"session_id"`
	ToolCallID         string     `json:"tool_call_id"`
	Questions          []Question `json:"questions"`
	ConfirmTitle       string     `json:"confirm_title,omitempty"`
	ConfirmDescription string     `json:"confirm_description,omitempty"`
}

// Validate checks that a Request has valid fields.
func (r Request) Validate() error {
	if len(r.Questions) == 0 {
		return fmt.Errorf("at least one question is required")
	}
	if len(r.Questions) > MaxQuestions {
		return fmt.Errorf("questions exceed maximum of %d (got %d)", MaxQuestions, len(r.Questions))
	}
	for i, q := range r.Questions {
		if err := q.Validate(); err != nil {
			return fmt.Errorf("question %d: %w", i+1, err)
		}
	}
	return nil
}

// Validate checks that a Question has valid fields.
func (q Question) Validate() error {
	label := q.identifier()
	if q.Text == "" {
		return fmt.Errorf("%s: question text is required", label)
	}
	if len(q.Text) > MaxQuestionLength {
		return fmt.Errorf("%s: text exceeds %d characters (got %d)", label, MaxQuestionLength, len(q.Text))
	}
	if q.Description == "" {
		return fmt.Errorf("%s: description is required", label)
	}
	if len(q.Description) > MaxDescriptionLength {
		return fmt.Errorf("%s: description exceeds %d characters (got %d)", label, MaxDescriptionLength, len(q.Description))
	}
	switch q.Type {
	case TypeYesNo, TypeFreeText:
	case TypeSingleChoice, TypeMultiChoice:
		if len(q.Choices) < 2 {
			return fmt.Errorf("%s: %s requires at least 2 choices in the \"choices\" array (got %d). Use \"choices\", not \"options\"", label, q.Type, len(q.Choices))
		}
		if len(q.Choices) > MaxChoices {
			return fmt.Errorf("%s: choices exceed maximum of %d (got %d)", label, MaxChoices, len(q.Choices))
		}
		seen := make(map[string]bool, len(q.Choices))
		for i, c := range q.Choices {
			if c.ID == "" {
				return fmt.Errorf("%s: choice %d must have an \"id\" field", label, i+1)
			}
			if seen[c.ID] {
				return fmt.Errorf("%s: choice %d has duplicate id %q", label, i+1, c.ID)
			}
			seen[c.ID] = true
			if c.Label == "" {
				return fmt.Errorf("%s: choice %d (%s) must have a \"label\" field", label, i+1, c.ID)
			}
			if len(c.Label) > MaxChoiceLabelLength {
				return fmt.Errorf("%s: choice %d label exceeds %d characters (got %d)", label, i+1, MaxChoiceLabelLength, len(c.Label))
			}
			if len(c.Description) > MaxChoiceDescriptionLength {
				return fmt.Errorf("%s: choice %d description exceeds %d characters (got %d)", label, i+1, MaxChoiceDescriptionLength, len(c.Description))
			}
		}
	default:
		return fmt.Errorf("%s: unknown type %q (must be yes_no, single_choice, multi_choice, or free_text)", label, q.Type)
	}
	return nil
}

func (q Question) identifier() string {
	if q.Label != "" {
		return fmt.Sprintf("[%s]", q.Label)
	}
	if q.Text != "" {
		text := q.Text
		if len(text) > 40 {
			text = text[:40] + "..."
		}
		return fmt.Sprintf("[%s]", text)
	}
	return "[unnamed question]"
}

const (
	MaxQuestionLength          = 240
	MaxDescriptionLength       = 600
	MaxChoiceLabelLength       = 200
	MaxChoiceDescriptionLength = 200
	MaxChoices                 = 5
	MaxQuestions               = 5
)

// Notification is published when a question batch is resolved.
type Notification struct {
	BatchID string `json:"batch_id"`
}

// Service manages question request lifecycles.
type Service interface {
	pubsub.Subscriber[Request]

	SubscribeNotifications(ctx context.Context) <-chan pubsub.Event[Notification]
	Ask(ctx context.Context, req Request) ([]Answer, error)
	Answer(batchID string, answers []Answer) bool
	Cancel(batchID string) bool
}

type result struct {
	answers []Answer
	err     error
}

type pendingRequest struct {
	respCh chan result
}

type service struct {
	broker             *pubsub.Broker[Request]
	notificationBroker *pubsub.Broker[Notification]
	pending            *csync.Map[string, *pendingRequest]
	mu                 sync.Mutex
}

// NewService creates a new question service.
func NewService() Service {
	return &service{
		broker:             pubsub.NewBroker[Request](),
		notificationBroker: pubsub.NewBroker[Notification](),
		pending:            csync.NewMap[string, *pendingRequest](),
	}
}

// Subscribe returns a channel for question events.
func (s *service) Subscribe(ctx context.Context) <-chan pubsub.Event[Request] {
	return s.broker.Subscribe(ctx)
}

// SubscribeNotifications returns a channel for resolution notifications.
func (s *service) SubscribeNotifications(ctx context.Context) <-chan pubsub.Event[Notification] {
	return s.notificationBroker.Subscribe(ctx)
}

// Ask publishes a request and blocks until it is answered or cancelled.
func (s *service) Ask(ctx context.Context, req Request) ([]Answer, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	for i := range req.Questions {
		if req.Questions[i].ID == "" {
			req.Questions[i].ID = uuid.NewString()
		}
	}
	if len(req.Questions) >= 2 {
		if req.ConfirmTitle == "" {
			req.ConfirmTitle = "Ready to go?"
		}
		if req.ConfirmDescription == "" {
			req.ConfirmDescription = "Review your answers above and confirm."
		}
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	entry := &pendingRequest{respCh: make(chan result, 1)}
	s.pending.Set(req.ID, entry)

	s.broker.Publish(pubsub.CreatedEvent, req)

	select {
	case res := <-entry.respCh:
		return res.answers, res.err
	case <-ctx.Done():
		if s.resolve(req.ID, result{err: ctx.Err()}) {
			return nil, ctx.Err()
		}
		res := <-entry.respCh
		return res.answers, res.err
	}
}

// Answer resolves the pending question identified by batchID.
func (s *service) Answer(batchID string, answers []Answer) bool {
	return s.resolve(batchID, result{answers: answers})
}

// Cancel resolves the pending question identified by batchID as cancelled.
func (s *service) Cancel(batchID string) bool {
	return s.resolve(batchID, result{err: ErrCancelled})
}

func (s *service) resolve(batchID string, res result) bool {
	if batchID == "" {
		return false
	}

	entry, ok := s.pending.Take(batchID)
	if !ok {
		return false
	}

	entry.respCh <- res
	s.notificationBroker.Publish(pubsub.CreatedEvent, Notification{BatchID: batchID})
	return true
}
