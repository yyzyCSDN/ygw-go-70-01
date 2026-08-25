package inverter

import (
	"errors"
	"testing"
)

func TestStateTransitions(t *testing.T) {
	if !CanTransition(StateStopped, StateStarting) {
		t.Fatal("stopped to starting must be allowed")
	}
	if CanTransition(StateStopped, StateRunning) {
		t.Fatal("stopped to running must be rejected")
	}
	if !CanTransition(StateFault, StateStopped) {
		t.Fatal("fault to stopped must be allowed")
	}
	if !ValidState(StateDerating) {
		t.Fatal("derating must be a valid state")
	}
}

func TestTableRegisterAndModelOf(t *testing.T) {
	table := NewTable()
	if err := table.Register(&Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := table.Register(&Inverter{ID: "inv1", Serial: "S2", Model: "PV-50K"}); err == nil {
		t.Fatal("duplicate register must fail")
	}
	model, err := table.ModelOf("inv1")
	if err != nil {
		t.Fatalf("model of: %v", err)
	}
	if model != "PV-100K" {
		t.Fatalf("model = %s, want PV-100K", model)
	}
	state, err := table.State("inv1")
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if state != StateStopped {
		t.Fatalf("initial state = %s, want stopped", state)
	}
}

func TestCatalogDefaults(t *testing.T) {
	catalog := DefaultCatalog()
	if !catalog.Has("PV-100K") {
		t.Fatal("PV-100K missing from default catalog")
	}
	model, err := catalog.Get("PV-50K")
	if err != nil {
		t.Fatalf("get PV-50K: %v", err)
	}
	if model.RatedPowerW != 50000 {
		t.Fatalf("rated power = %d, want 50000", model.RatedPowerW)
	}
	if len(catalog.List()) != 3 {
		t.Fatalf("catalog size = %d, want 3", len(catalog.List()))
	}
}

// TestMarkGridResultReflectsError is the regression for the swallowed-error
// bug: passing an error must leave the inverter off-grid and record it, while
// a nil error must mark it on-grid. The old implementation ignored err and
// always marked on-grid.
func TestMarkGridResultReflectsError(t *testing.T) {
	table := NewTable()
	if err := table.Register(&Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	gridErr := errors.New("pvcontrol: grid sync failed")
	if err := table.MarkGridResult("inv1", gridErr); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	inv, _ := table.Get("inv1")
	if inv.GridState != GridOffGrid {
		t.Fatalf("grid state after error = %s, want off-grid", inv.GridState)
	}
	if inv.LastGridErr != gridErr.Error() {
		t.Fatalf("last grid err = %q, want %q", inv.LastGridErr, gridErr.Error())
	}
	if inv.GridErrCount != 1 {
		t.Fatalf("grid err count = %d, want 1", inv.GridErrCount)
	}

	// A subsequent success must clear the error and move on-grid.
	if err := table.MarkGridResult("inv1", nil); err != nil {
		t.Fatalf("mark success: %v", err)
	}
	inv, _ = table.Get("inv1")
	if inv.GridState != GridOnGrid {
		t.Fatalf("grid state after success = %s, want on-grid", inv.GridState)
	}
	if inv.LastGridErr != "" {
		t.Fatalf("last grid err after success = %q, want empty", inv.LastGridErr)
	}
}
