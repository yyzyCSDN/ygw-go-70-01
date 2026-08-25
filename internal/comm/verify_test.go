package comm

import (
	"errors"
	"testing"

	"pvcontrol/internal/inverter"
)

type ackHandle struct {
	id      string
	replied bool
}

func (h *ackHandle) ID() string {
	return h.id
}

func (h *ackHandle) Write(payload []byte) error {
	return nil
}

func (h *ackHandle) Read() ([]byte, error) {
	if h.replied {
		return nil, errors.New("no more replies")
	}
	h.replied = true
	return EncodeFrame(&inverter.Frame{Serial: "S1", Kind: inverter.FrameAck, Seq: 1, Payload: []byte("ok")}), nil
}

func (h *ackHandle) Close() error {
	return nil
}

type ackTransport struct{}

func (ackTransport) Open(invID string) (Handle, error) {
	return &ackHandle{id: invID}, nil
}

func TestCommQueueNotOverwrittenByStaleBatch(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	client := NewClient(table, ackTransport{})
	batch := []inverter.QueuedMessage{
		{InverterID: "inv1", Seq: 1, Kind: "limit", Payload: inverter.EncodePower(40000)},
		{InverterID: "inv1", Seq: 2, Kind: "limit", Payload: inverter.EncodePower(30000)},
		{InverterID: "inv1", Seq: 3, Kind: "param", Payload: []byte("cfg")},
	}
	results, err := client.ProcessBatch(batch)
	if err != nil {
		t.Fatalf("process batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for _, seq := range []int{1, 2, 3} {
		status, err := table.MessageStatus("inv1", seq)
		if err != nil {
			t.Fatalf("status %d: %v", seq, err)
		}
		if status.Status != inverter.MessageSent {
			t.Fatalf("seq %d status = %s, want sent", seq, status.Status)
		}
	}
}
