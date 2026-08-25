package grid

import (
	"testing"

	"pvcontrol/internal/comm"
	"pvcontrol/internal/inverter"
)

type silentComm struct{}

func (silentComm) SyncGrid(invID string) error {
	return nil
}

func (silentComm) DispatchSequence(invID string, seq comm.SwitchSequence) error {
	return nil
}

func TestManagerDefaults(t *testing.T) {
	table := inverter.NewTable()
	manager := NewManager(table, silentComm{})
	if manager.StateOf("inv1") != StateOffGrid {
		t.Fatal("default grid state must be off-grid")
	}
	if manager.MemberCount() != 0 {
		t.Fatal("new manager must have no members")
	}
}

func TestSequenceBuild(t *testing.T) {
	table := inverter.NewTable()
	manager := NewManager(table, silentComm{})
	seq := manager.buildSequence("A", StateOnGrid)
	if seq.Version != 1 {
		t.Fatalf("first sequence version = %d, want 1", seq.Version)
	}
	if seq.StepCount() != 0 {
		t.Fatalf("empty membership sequence steps = %d, want 0", seq.StepCount())
	}
	_ = table.Register(&inverter.Inverter{ID: "B", Serial: "S2", Model: "PV-50K"})
	_ = manager.Join("B")
	seq2 := manager.buildSequence("B", StateOnGrid)
	if seq2.StepCount() != 1 || !seq2.HasStep("B") {
		t.Fatalf("sequence after join = %+v", seq2)
	}
}
