package comm

import (
	"testing"

	"pvcontrol/internal/inverter"
)

func TestCommHandleClosed(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	_ = table.Register(&inverter.Inverter{ID: "inv2", Serial: "S2", Model: "PV-50K"})
	client := NewClient(table, NewSimTransport(table))
	if err := client.Patrol([]string{"inv1", "inv2"}); err != nil {
		t.Fatalf("patrol: %v", err)
	}
	for _, id := range []string{"inv1", "inv2"} {
		active, err := table.ActiveLeases(id)
		if err != nil {
			t.Fatalf("leases %s: %v", id, err)
		}
		if active != 0 {
			t.Fatalf("active leases for %s = %d, want 0", id, active)
		}
	}
	if got := client.SessionCount(); got != 0 {
		t.Fatalf("sessions = %d, want 0", got)
	}
}
