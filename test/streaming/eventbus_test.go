package streaming_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/streaming"
)

func TestEventBus_BasicSubscription(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create a subscriber
	ch := make(chan streaming.StreamEvent, 10)
	bus.Subscribe("job1", "stdout", ch)

	// Check subscriber count
	if count := bus.GetSubscriberCount("job1", "stdout"); count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}

	// Check if has subscribers
	if !bus.HasSubscribers("job1", "stdout") {
		t.Error("Expected to have subscribers")
	}

	// Check different job/output has no subscribers
	if bus.HasSubscribers("job2", "stdout") {
		t.Error("Expected job2 to have no subscribers")
	}

	if bus.HasSubscribers("job1", "stderr") {
		t.Error("Expected job1 stderr to have no subscribers")
	}
}

func TestEventBus_PublishEvent(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create subscriber
	ch := make(chan streaming.StreamEvent, 10)
	bus.Subscribe("job1", "stdout", ch)

	// Publish event
	event := &streaming.StreamEvent{
		JobID:      "job1",
		OutputType: "stdout",
		Content:    "test content",
		EventType:  streaming.StreamEventNewOutput,
		Timestamp:  time.Now(),
	}

	bus.Publish(event)

	// Check event was received
	select {
	case receivedEvent := <-ch:
		if receivedEvent.JobID != event.JobID {
			t.Errorf("Expected JobID %s, got %s", event.JobID, receivedEvent.JobID)
		}
		if receivedEvent.Content != event.Content {
			t.Errorf("Expected content %s, got %s", event.Content, receivedEvent.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Expected to receive event within 1 second")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create multiple subscribers
	ch1 := make(chan streaming.StreamEvent, 10)
	ch2 := make(chan streaming.StreamEvent, 10)
	ch3 := make(chan streaming.StreamEvent, 10)

	bus.Subscribe("job1", "stdout", ch1)
	bus.Subscribe("job1", "stdout", ch2)
	bus.Subscribe("job1", "stdout", ch3)

	if count := bus.GetSubscriberCount("job1", "stdout"); count != 3 {
		t.Errorf("Expected 3 subscribers, got %d", count)
	}

	// Publish event
	event := &streaming.StreamEvent{
		JobID:      "job1",
		OutputType: "stdout",
		Content:    "broadcast test",
		EventType:  streaming.StreamEventNewOutput,
		Timestamp:  time.Now(),
	}

	bus.Publish(event)

	// All subscribers should receive the event
	channels := []chan streaming.StreamEvent{ch1, ch2, ch3}
	for i, ch := range channels {
		select {
		case receivedEvent := <-ch:
			if receivedEvent.Content != event.Content {
				t.Errorf("Subscriber %d: expected content %s, got %s", i, event.Content, receivedEvent.Content)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Subscriber %d: expected to receive event within 1 second", i)
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create subscribers
	ch1 := make(chan streaming.StreamEvent, 10)
	ch2 := make(chan streaming.StreamEvent, 10)

	bus.Subscribe("job1", "stdout", ch1)
	bus.Subscribe("job1", "stdout", ch2)

	if count := bus.GetSubscriberCount("job1", "stdout"); count != 2 {
		t.Errorf("Expected 2 subscribers, got %d", count)
	}

	// Unsubscribe one
	bus.Unsubscribe("job1", "stdout", ch1)

	if count := bus.GetSubscriberCount("job1", "stdout"); count != 1 {
		t.Errorf("Expected 1 subscriber after unsubscribe, got %d", count)
	}

	// Publish event - only ch2 should receive it
	event := &streaming.StreamEvent{
		JobID:      "job1",
		OutputType: "stdout",
		Content:    "after unsubscribe",
		EventType:  streaming.StreamEventNewOutput,
		Timestamp:  time.Now(),
	}

	bus.Publish(event)

	// ch1 should not receive the event
	select {
	case <-ch1:
		t.Error("ch1 should not receive event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event received
	}

	// ch2 should receive the event
	select {
	case receivedEvent := <-ch2:
		if receivedEvent.Content != event.Content {
			t.Errorf("Expected content %s, got %s", event.Content, receivedEvent.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("ch2 should receive event")
	}
}

func TestEventBus_UnsubscribeAll(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create multiple subscribers
	ch1 := make(chan streaming.StreamEvent, 10)
	ch2 := make(chan streaming.StreamEvent, 10)
	ch3 := make(chan streaming.StreamEvent, 10)

	bus.Subscribe("job1", "stdout", ch1)
	bus.Subscribe("job1", "stdout", ch2)
	bus.Subscribe("job1", "stderr", ch3) // Different output type

	// Unsubscribe all for job1/stdout
	bus.UnsubscribeAll("job1", "stdout")

	if count := bus.GetSubscriberCount("job1", "stdout"); count != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe all, got %d", count)
	}

	// stderr subscriber should remain
	if count := bus.GetSubscriberCount("job1", "stderr"); count != 1 {
		t.Errorf("Expected stderr subscriber to remain, got %d", count)
	}

	// Channels should be closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("Expected ch1 to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected ch1 to be closed immediately")
	}

	select {
	case _, ok := <-ch2:
		if ok {
			t.Error("Expected ch2 to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected ch2 to be closed immediately")
	}
}

func TestEventBus_PublishError(t *testing.T) {
	bus := streaming.NewEventBus()

	ch := make(chan streaming.StreamEvent, 10)
	bus.Subscribe("job1", "stdout", ch)

	// Publish error
	testErr := fmt.Errorf("test error")
	bus.PublishError("job1", "stdout", testErr)

	// Check error event was received
	select {
	case event := <-ch:
		if event.EventType != streaming.StreamEventError {
			t.Errorf("Expected error event type, got %v", event.EventType)
		}
		if event.Error == nil {
			t.Error("Expected error to be set")
		}
		if event.Error.Error() != testErr.Error() {
			t.Errorf("Expected error %v, got %v", testErr, event.Error)
		}
	case <-time.After(1 * time.Second):
		t.Error("Expected to receive error event")
	}
}

func TestEventBus_GetAllSubscriptions(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create various subscriptions
	ch1 := make(chan streaming.StreamEvent, 10)
	ch2 := make(chan streaming.StreamEvent, 10)
	ch3 := make(chan streaming.StreamEvent, 10)

	bus.Subscribe("job1", "stdout", ch1)
	bus.Subscribe("job1", "stdout", ch2)
	bus.Subscribe("job1", "stderr", ch3)
	bus.Subscribe("job2", "stdout", ch1) // Same channel, different job

	subscriptions := bus.GetAllSubscriptions()

	expectedKeys := map[string]int{
		"job1:stdout": 2,
		"job1:stderr": 1,
		"job2:stdout": 1,
	}

	if len(subscriptions) != len(expectedKeys) {
		t.Errorf("Expected %d subscription keys, got %d", len(expectedKeys), len(subscriptions))
	}

	for key, expectedCount := range expectedKeys {
		if count, exists := subscriptions[key]; !exists {
			t.Errorf("Expected subscription key %s to exist", key)
		} else if count != expectedCount {
			t.Errorf("Expected %d subscribers for %s, got %d", expectedCount, key, count)
		}
	}
}

func TestEventBus_GetSubscriptionInfo(t *testing.T) {
	bus := streaming.NewEventBus()

	ch1 := make(chan streaming.StreamEvent, 10)
	ch2 := make(chan streaming.StreamEvent, 10)

	bus.Subscribe("job1", "stdout", ch1)
	bus.Subscribe("job2", "stderr", ch2)

	info := bus.GetSubscriptionInfo()

	if len(info) != 2 {
		t.Errorf("Expected 2 subscription info entries, got %d", len(info))
	}

	// Find job1 info
	var job1Info *streaming.SubscriptionInfo
	for i := range info {
		if info[i].JobID == "job1" {
			job1Info = &info[i]
			break
		}
	}

	if job1Info == nil {
		t.Error("Expected to find job1 subscription info")
	} else {
		if job1Info.OutputType != "stdout" {
			t.Errorf("Expected stdout, got %s", job1Info.OutputType)
		}
		if job1Info.Subscribers != 1 {
			t.Errorf("Expected 1 subscriber, got %d", job1Info.Subscribers)
		}
	}
}

func TestEventBus_Clear(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create subscriptions
	ch1 := make(chan streaming.StreamEvent, 10)
	ch2 := make(chan streaming.StreamEvent, 10)

	bus.Subscribe("job1", "stdout", ch1)
	bus.Subscribe("job2", "stderr", ch2)

	if len(bus.GetAllSubscriptions()) == 0 {
		t.Error("Expected subscriptions before clear")
	}

	// Clear all
	bus.Clear()

	// Check all subscriptions are gone
	subscriptions := bus.GetAllSubscriptions()
	if len(subscriptions) != 0 {
		t.Errorf("Expected 0 subscriptions after clear, got %d", len(subscriptions))
	}

	// Channels should be closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("Expected ch1 to be closed after clear")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected ch1 to be closed immediately")
	}
}

func TestEventBus_ThreadSafety(t *testing.T) {
	bus := streaming.NewEventBus()

	var wg sync.WaitGroup
	numRoutines := 10
	operationsPerRoutine := 50

	// Start multiple routines doing various operations
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			ch := make(chan streaming.StreamEvent, 100)
			jobID := fmt.Sprintf("job%d", routineID%3) // Use 3 different jobs

			for j := 0; j < operationsPerRoutine; j++ {
				// Subscribe
				bus.Subscribe(jobID, "stdout", ch)

				// Publish event
				event := &streaming.StreamEvent{
					JobID:      jobID,
					OutputType: "stdout",
					Content:    fmt.Sprintf("routine%d-msg%d", routineID, j),
					EventType:  streaming.StreamEventNewOutput,
					Timestamp:  time.Now(),
				}
				bus.Publish(event)

				// Check subscriber count
				_ = bus.GetSubscriberCount(jobID, "stdout")

				// Get subscription info
				_ = bus.GetAllSubscriptions()

				// Occasionally unsubscribe
				if j%10 == 0 {
					bus.Unsubscribe(jobID, "stdout", ch)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	subscriptions := bus.GetAllSubscriptions()
	totalSubscribers := 0
	for _, count := range subscriptions {
		totalSubscribers += count
	}

	t.Logf("Final state: %d subscription keys, %d total subscribers", len(subscriptions), totalSubscribers)
}

func TestEventBus_FullChannelHandling(t *testing.T) {
	bus := streaming.NewEventBus()

	// Create channel with small buffer
	ch := make(chan streaming.StreamEvent, 1)
	bus.Subscribe("job1", "stdout", ch)

	// Fill the channel
	event1 := &streaming.StreamEvent{
		JobID:      "job1",
		OutputType: "stdout",
		Content:    "message1",
		EventType:  streaming.StreamEventNewOutput,
		Timestamp:  time.Now(),
	}
	bus.Publish(event1)

	// Publish another event - should not block even if channel is full
	event2 := &streaming.StreamEvent{
		JobID:      "job1",
		OutputType: "stdout",
		Content:    "message2",
		EventType:  streaming.StreamEventNewOutput,
		Timestamp:  time.Now(),
	}

	// This should complete quickly even with full channel
	done := make(chan bool)
	go func() {
		bus.Publish(event2)
		done <- true
	}()

	select {
	case <-done:
		// Good - publish didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("Publish blocked on full channel")
	}

	// Verify first event was received
	select {
	case receivedEvent := <-ch:
		if receivedEvent.Content != event1.Content {
			t.Errorf("Expected first event content %s, got %s", event1.Content, receivedEvent.Content)
		}
	default:
		t.Error("Expected to receive first event")
	}
}
