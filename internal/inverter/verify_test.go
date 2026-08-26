package inverter

import "testing"

func TestInverterStateUsesLatestModel(t *testing.T) {
	table := NewTable()
	if err := table.Register(&Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := table.Replace("inv1", "S1-NEW", "PV-200K"); err != nil {
		t.Fatalf("replace: %v", err)
	}
	model, err := table.ModelOf("inv1")
	if err != nil {
		t.Fatalf("model of: %v", err)
	}
	if model != "PV-200K" {
		t.Fatalf("model = %s, want PV-200K", model)
	}
	inv, err := table.Get("inv1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if inv.Model != "PV-200K" {
		t.Fatalf("record model = %s, want PV-200K", inv.Model)
	}
}
