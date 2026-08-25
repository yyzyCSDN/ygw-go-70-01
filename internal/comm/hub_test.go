package comm

import (
	"errors"
	"testing"

	"pvcontrol/internal/inverter"
)

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	client := NewClient(table, NewSimTransport(table))
	hub := NewHub()
	if err := hub.Add("main", client); err != nil {
		t.Fatalf("hub add: %v", err)
	}
	return hub
}

// TestHubIngestWithEmptyFrame guards against the panic regression: an empty
// frame must produce an explicit error result, not a nil frame with a nil
// error that callers would dereference.
func TestHubIngestWithEmptyFrame(t *testing.T) {
	hub := newTestHub(t)

	frame, err := hub.IngestWith("main", []byte{})
	if err == nil {
		t.Fatalf("IngestWith(empty) err = nil, want non-nil; frame=%v", frame)
	}
	if !errors.Is(err, ErrEmptyFrame) {
		t.Fatalf("IngestWith(empty) err = %v, want ErrEmptyFrame", err)
	}
	if frame != nil {
		t.Fatalf("IngestWith(empty) frame = %+v, want nil", frame)
	}
}

// TestHubIngestWithUnknownClient ensures an unknown hub client still errors
// before any parsing is attempted.
func TestHubIngestWithUnknownClient(t *testing.T) {
	hub := newTestHub(t)
	if _, err := hub.IngestWith("nope", []byte{}); err == nil {
		t.Fatal("IngestWith(unknown client) err = nil, want non-nil")
	}
}

// TestHubIngestWithValidFrame confirms the happy path still returns the frame.
func TestHubIngestWithValidFrame(t *testing.T) {
	hub := newTestHub(t)
	raw := EncodeFrame(&inverter.Frame{Serial: "S1", Kind: inverter.FrameStatus, Seq: 1, Payload: []byte("ok")})

	frame, err := hub.IngestWith("main", raw)
	if err != nil {
		t.Fatalf("IngestWith(valid) err = %v, want nil", err)
	}
	if frame == nil {
		t.Fatal("IngestWith(valid) frame = nil, want non-nil")
	}
	if frame.Serial != "S1" || frame.Kind != inverter.FrameStatus {
		t.Fatalf("IngestWith(valid) frame = %+v", frame)
	}
}
