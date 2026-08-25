package comm

import (
	"bytes"
	"testing"

	"pvcontrol/internal/inverter"
)

func TestFrameRoundTrip(t *testing.T) {
	frame := &inverter.Frame{
		Serial:  "PV1A0001",
		Kind:    inverter.FrameLimit,
		Seq:     7,
		Payload: []byte{0, 1, 0x86, 0xa0},
	}
	raw := EncodeFrame(frame)
	parsed, err := ParseFrame(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Serial != frame.Serial || parsed.Kind != frame.Kind || parsed.Seq != frame.Seq {
		t.Fatalf("parsed frame mismatch: %+v", parsed)
	}
	if !bytes.Equal(parsed.Payload, frame.Payload) {
		t.Fatalf("payload mismatch: %v", parsed.Payload)
	}
}

func TestFrameCorruptionDetected(t *testing.T) {
	frame := &inverter.Frame{Serial: "S1", Kind: inverter.FrameStatus, Seq: 1, Payload: []byte("ok")}
	raw := EncodeFrame(frame)
	raw[len(raw)-1] ^= 0xff
	if _, err := ParseFrame(raw); err == nil {
		t.Fatal("corrupted frame must fail checksum validation")
	}
}

func TestSimTransportExchange(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	transport := NewSimTransport(table)
	handle, err := transport.Open("inv1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	frame := &inverter.Frame{Serial: "S1", Kind: inverter.FrameStatus, Seq: 3, Payload: nil}
	if err := handle.Write(EncodeFrame(frame)); err != nil {
		t.Fatalf("write: %v", err)
	}
	reply, err := handle.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	ack, err := ParseFrame(reply)
	if err != nil {
		t.Fatalf("parse ack: %v", err)
	}
	if ack.Kind != inverter.FrameStatus {
		t.Fatalf("ack kind = %s, want status", ack.Kind)
	}
	_ = handle.Close()
}
