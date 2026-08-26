package inverter

import (
	"errors"
	"testing"
)

type failingPort struct {
	writebackErr error
	dedupMarked  bool
	cleared      bool
}

func (f *failingPort) SendCommand(invID string, cmd Command, payload []byte) error {
	return nil
}

func (f *failingPort) WritebackState(invID string, state State) error {
	return f.writebackErr
}

func (f *failingPort) CheckDedup(invID string, cmd Command, payload []byte) bool {
	if cmd == CommandLimit {
		return f.dedupMarked
	}
	return false
}

func (f *failingPort) MarkDedup(invID string, cmd Command, payload []byte) {
	if cmd == CommandLimit {
		f.dedupMarked = true
	}
}

func (f *failingPort) ClearDedup(invID string, cmd Command) error {
	if cmd == CommandLimit {
		f.cleared = true
		f.dedupMarked = false
	}
	return nil
}

func TestInverterRecoveryWritebackErrorNotSwallowed(t *testing.T) {
	table := NewTable()
	if err := table.Register(&Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := table.SetState("inv1", StateFault); err != nil {
		t.Fatalf("fault: %v", err)
	}
	port := &failingPort{writebackErr: errors.New("writeback failed"), dedupMarked: true}
	station := NewStation(table, port)
	if err := station.Recover("inv1"); err == nil {
		t.Fatal("recover must return the writeback error")
	}
	if port.dedupMarked {
		t.Fatal("limit-power dedup key must be cleared after recovery failure")
	}
}
