package alarm

import (
	"testing"

	"pvcontrol/internal/inverter"
)

func TestRaiseResolve(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	manager := NewManager(table)
	if err := manager.Raise("inv1", "overvoltage"); err != nil {
		t.Fatalf("raise: %v", err)
	}
	if manager.ActiveCount() != 1 {
		t.Fatalf("active = %d, want 1", manager.ActiveCount())
	}
	if err := manager.Resolve("inv1"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if manager.ActiveCount() != 0 {
		t.Fatalf("active after resolve = %d, want 0", manager.ActiveCount())
	}
}

func TestRaiseSetsFaultState(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	manager := NewManager(table)
	if err := manager.Raise("inv1", "overtemp"); err != nil {
		t.Fatalf("raise: %v", err)
	}
	state, err := table.State("inv1")
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if state != inverter.StateFault {
		t.Fatalf("state = %s, want fault", state)
	}
}
