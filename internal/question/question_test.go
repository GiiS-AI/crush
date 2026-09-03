package question

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GiiS-AI/GiiS-Code/internal/pubsub"
	"github.com/stretchr/testify/require"
)

func waitForQuestionEvent(t *testing.T, ch <-chan pubsub.Event[Request]) pubsub.Event[Request] {
	t.Helper()
	select {
	case evt := <-ch:
		return evt
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for question request event")
		return pubsub.Event[Request]{}
	}
}

func waitForQuestionNotification(t *testing.T, ch <-chan pubsub.Event[Notification]) pubsub.Event[Notification] {
	t.Helper()
	select {
	case evt := <-ch:
		return evt
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for question notification")
		return pubsub.Event[Notification]{}
	}
}

func TestAskSupportsIndependentConcurrentBatches(t *testing.T) {
	t.Parallel()

	svc := NewService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqCh := svc.Subscribe(ctx)

	type askResult struct {
		answers []Answer
		err     error
	}

	firstDone := make(chan askResult, 1)
	secondDone := make(chan askResult, 1)

	go func() {
		answers, err := svc.Ask(context.Background(), Request{
			ID: "batch-1",
			Questions: []Question{{
				ID:          "q-1",
				Type:        TypeYesNo,
				Text:        "Ship batch one?",
				Description: "Pick yes or no.",
			}},
		})
		firstDone <- askResult{answers: answers, err: err}
	}()
	go func() {
		answers, err := svc.Ask(context.Background(), Request{
			ID: "batch-2",
			Questions: []Question{{
				ID:          "q-2",
				Type:        TypeFreeText,
				Text:        "Name batch two",
				Description: "Provide a short answer.",
			}},
		})
		secondDone <- askResult{answers: answers, err: err}
	}()

	seen := map[string]bool{}
	for len(seen) < 2 {
		evt := waitForQuestionEvent(t, reqCh)
		require.Equal(t, pubsub.CreatedEvent, evt.Type)
		seen[evt.Payload.ID] = true
	}

	require.True(t, svc.Answer("batch-2", []Answer{{QuestionID: "q-2", FillInText: "beta"}}))

	select {
	case res := <-secondDone:
		require.NoError(t, res.err)
		require.Equal(t, []Answer{{QuestionID: "q-2", FillInText: "beta"}}, res.answers)
	case <-time.After(time.Second):
		t.Fatal("second batch did not resolve")
	}

	select {
	case <-firstDone:
		t.Fatal("first batch resolved before it was answered")
	case <-time.After(50 * time.Millisecond):
	}

	require.True(t, svc.Answer("batch-1", []Answer{{QuestionID: "q-1", Yes: boolPtr(true)}}))

	select {
	case res := <-firstDone:
		require.NoError(t, res.err)
		require.Len(t, res.answers, 1)
		require.NotNil(t, res.answers[0].Yes)
		require.True(t, *res.answers[0].Yes)
	case <-time.After(time.Second):
		t.Fatal("first batch did not resolve")
	}
}

func TestCancelPublishesNotificationAndUnblocksAsk(t *testing.T) {
	t.Parallel()

	svc := NewService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqCh := svc.Subscribe(ctx)
	notificationCh := svc.SubscribeNotifications(ctx)

	done := make(chan error, 1)
	go func() {
		_, err := svc.Ask(context.Background(), Request{
			ID: "batch-cancel",
			Questions: []Question{{
				ID:          "q-cancel",
				Type:        TypeYesNo,
				Text:        "Proceed?",
				Description: "Pick yes or no.",
			}},
		})
		done <- err
	}()

	_ = waitForQuestionEvent(t, reqCh)
	require.True(t, svc.Cancel("batch-cancel"))

	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrCancelled)
	case <-time.After(time.Second):
		t.Fatal("Ask did not unblock after cancel")
	}

	evt := waitForQuestionNotification(t, notificationCh)
	require.Equal(t, pubsub.CreatedEvent, evt.Type)
	require.Equal(t, "batch-cancel", evt.Payload.BatchID)
	require.False(t, svc.Answer("batch-cancel", []Answer{{QuestionID: "q-cancel"}}))
}

func TestAskContextCancelResolvesPendingBatch(t *testing.T) {
	t.Parallel()

	svc := NewService()
	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	reqCh := svc.Subscribe(ctx)
	notificationCh := svc.SubscribeNotifications(ctx)

	askCtx, cancelAsk := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := svc.Ask(askCtx, Request{
			ID: "batch-timeout",
			Questions: []Question{{
				ID:          "q-timeout",
				Type:        TypeFreeText,
				Text:        "Why?",
				Description: "Provide context.",
			}},
		})
		done <- err
	}()

	_ = waitForQuestionEvent(t, reqCh)
	cancelAsk()

	select {
	case err := <-done:
		require.True(t, errors.Is(err, context.Canceled))
	case <-time.After(time.Second):
		t.Fatal("Ask did not return after context cancellation")
	}

	evt := waitForQuestionNotification(t, notificationCh)
	require.Equal(t, "batch-timeout", evt.Payload.BatchID)
	require.False(t, svc.Cancel("batch-timeout"))
}

func boolPtr(v bool) *bool {
	return &v
}
