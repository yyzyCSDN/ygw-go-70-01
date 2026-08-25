package inverter

import "testing"

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
