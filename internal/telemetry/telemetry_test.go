package telemetry

import (
	"testing"
	"time"
)

func TestMetricsRecordAndSnapshot(t *testing.T) {
	m := NewMetrics()

	m.Record(true, true, 250, 400, true, true, false, false, false, false)
	m.Record(false, true, 100, 0, false, false, false, false, false, false)
	m.Record(true, false, 500, 0, false, false, false, false, false, false)

	snap := m.Snapshot()

	if snap.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", snap.TotalRequests)
	}
	if snap.RepairedRequests != 2 {
		t.Errorf("expected 2 repaired requests, got %d", snap.RepairedRequests)
	}
	if snap.ValidPassThrough != 1 {
		t.Errorf("expected 1 valid pass-through, got %d", snap.ValidPassThrough)
	}
	if snap.FailedRepairs != 1 {
		t.Errorf("expected 1 failed repair, got %d", snap.FailedRepairs)
	}
	if snap.TokensSavedEstimated != 400 {
		t.Errorf("expected 400 tokens saved, got %d", snap.TokensSavedEstimated)
	}
	if snap.TruncationFixed != 1 {
		t.Errorf("expected 1 truncation fixed, got %d", snap.TruncationFixed)
	}
	if snap.MarkdownStripped != 1 {
		t.Errorf("expected 1 markdown stripped, got %d", snap.MarkdownStripped)
	}
}

func TestStoreInsertAndQuery(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory sqlite: %v", err)
	}
	defer store.Close()

	evt := &RepairEvent{
		TraceID:        "trc-101",
		Timestamp:      time.Now(),
		Provider:       "openai",
		Model:          "gpt-4o-mini",
		Status:         "healed",
		LatencyUs:      350,
		TokensSaved:    450,
		OriginalSize:   120,
		RepairedSize:   115,
		Flags:          "stripped_markdown, balanced_delimiters",
		RawInput:       "```json\n{\"test\": 1",
		RepairedOutput: "{\"test\": 1}",
	}

	if err := store.InsertEvent(evt); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	events, count, err := store.GetEvents(10, 0)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if count != 1 || len(events) != 1 {
		t.Fatalf("expected 1 event, got count %d, len %d", count, len(events))
	}

	if events[0].TraceID != "trc-101" || events[0].Status != "healed" {
		t.Errorf("mismatch in retrieved event: %+v", events[0])
	}
}

func TestEventBusPubSub(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe()
	defer bus.Unsubscribe(ch)

	evt := &RepairEvent{
		TraceID: "trc-sub",
		Status:  "healed",
	}

	bus.Publish(evt)

	select {
	case data := <-ch:
		if len(data) == 0 {
			t.Errorf("expected non-empty payload")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for event bus message")
	}
}
