package grid

import (
	"errors"
	"testing"

	"pvcontrol/internal/comm"
	"pvcontrol/internal/inverter"
)

type failingSync struct{}

func (failingSync) SyncGrid(invID string) error {
	return errors.New("sync failed")
}

func (failingSync) DispatchSequence(invID string, seq comm.SwitchSequence) error {
	return nil
}

func TestGridErrorNotSwallowed(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	manager := NewManager(table, failingSync{})
	if err := manager.Connect("inv1"); err == nil {
		t.Fatal("grid connect must return an error when sync fails")
	}
	state, err := table.GridStateOf("inv1")
	if err != nil {
		t.Fatalf("grid state: %v", err)
	}
	if state != inverter.GridOffGrid {
		t.Fatalf("grid state = %s, want off-grid", state)
	}
}
