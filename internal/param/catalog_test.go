package param

import (
	"testing"

	"pvcontrol/internal/inverter"
)

func TestCatalogForModel(t *testing.T) {
	catalog := NewCatalog()
	_ = catalog.Add("PV-100K", inverter.ParamSet{Model: "PV-100K", Rev: 1, LimitPowerW: 100000})
	set, err := catalog.ForModel("PV-100K")
	if err != nil {
		t.Fatalf("for model: %v", err)
	}
	if set.LimitPowerW != 100000 {
		t.Fatalf("limit power = %d, want 100000", set.LimitPowerW)
	}
	if _, err := catalog.ForModel("PV-999K"); err == nil {
		t.Fatal("unknown model must fail")
	}
}

func TestCatalogValidate(t *testing.T) {
	catalog := NewCatalog()
	_ = catalog.Add("PV-50K", inverter.ParamSet{Model: "PV-50K", Rev: 1, LimitPowerW: 50000})
	valid := inverter.ParamSet{Model: "PV-50K", LimitPowerW: 45000, RampRate: 20, ReactiveLimit: 30}
	if err := catalog.Validate(valid, "PV-50K"); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	invalid := inverter.ParamSet{Model: "PV-50K", LimitPowerW: 0}
	if err := catalog.Validate(invalid, "PV-50K"); err == nil {
		t.Fatal("zero limit power must be rejected")
	}
}
