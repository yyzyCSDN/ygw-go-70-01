package comm

import (
	"errors"
	"testing"

	"pvcontrol/internal/inverter"
)

func TestEmptyFrameNoNilPanic(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	client := NewClient(table, NewSimTransport(table))
	hub := NewHub()
	_ = hub.Add("main", client)
	frame, err := hub.IngestWith("main", []byte{})
	if err == nil {
		t.Fatalf("empty frame must return an error, got frame %+v", frame)
	}
	if !errors.Is(err, ErrEmptyFrame) {
		t.Fatalf("error = %v, want ErrEmptyFrame", err)
	}
}
