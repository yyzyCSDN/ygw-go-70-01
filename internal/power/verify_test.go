package power

import "testing"

type recordingDispatcher struct {
	targets []int
}

func (r *recordingDispatcher) DispatchTarget(targetW int) error {
	r.targets = append(r.targets, targetW)
	return nil
}

func TestPowerTargetUpdatedOnIrradiance(t *testing.T) {
	rec := &recordingDispatcher{}
	manager := NewManager(100000, rec)
	if err := manager.UpdateIrradiance(900); err != nil {
		t.Fatalf("update 900: %v", err)
	}
	if err := manager.Tick(); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if err := manager.UpdateIrradiance(400); err != nil {
		t.Fatalf("update 400: %v", err)
	}
	if got := manager.Target(); got != 40000 {
		t.Fatalf("target = %d, want 40000", got)
	}
	if len(rec.targets) != 2 || rec.targets[0] != 90000 || rec.targets[1] != 40000 {
		t.Fatalf("dispatched targets = %v, want [90000 40000]", rec.targets)
	}
}
