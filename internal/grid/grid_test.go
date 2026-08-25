package grid

import (
	"errors"
	"strings"
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

// failingComm fails every grid-sync attempt with syncErr, simulating an
// inverter that never acks (cable down, device offline).
type failingComm struct{ syncErr error }

func (f failingComm) SyncGrid(invID string) error { return f.syncErr }
func (failingComm) DispatchSequence(invID string, seq comm.SwitchSequence) error {
	return nil
}

// flakyComm succeeds only on the attempt indicated by successOn (1-based),
// failing every attempt before it.
type flakyComm struct {
	calls    int
	successOn int
}

func (f *flakyComm) SyncGrid(invID string) error {
	f.calls++
	if f.calls >= f.successOn {
		return nil
	}
	return errors.New("pvcontrol: transient sync drop")
}

func (f *flakyComm) DispatchSequence(invID string, seq comm.SwitchSequence) error {
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

// TestConnectSuccessMarksOnGrid guards the happy path: a successful sync must
// move the inverter to on-grid and clear any prior error.
func TestConnectSuccessMarksOnGrid(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	manager := NewManager(table, silentComm{})

	if err := manager.Connect("inv1"); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if got := manager.StateOf("inv1"); got != StateOnGrid {
		t.Fatalf("state = %s, want on-grid", got)
	}
	if err := manager.LastError("inv1"); err != nil {
		t.Fatalf("last error = %v, want nil", err)
	}
	gs, _ := table.GridStateOf("inv1")
	if gs != inverter.GridOnGrid {
		t.Fatalf("table grid state = %s, want on-grid", gs)
	}
}

// TestConnectFailureStaysOffGrid is the core regression for the swallowed-
// error bug: a failed sync must leave the inverter off-grid and RETURN the
// error, never report success. The previous implementation marked the
// inverter on-grid and returned nil.
func TestConnectFailureStaysOffGrid(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	syncErr := errors.New("pvcontrol: device not responding")
	manager := NewManager(table, failingComm{syncErr: syncErr})

	err := manager.Connect("inv1")
	if err == nil {
		t.Fatal("connect returned nil for a failed sync; failure must be reported, not swallowed")
	}
	if !errors.Is(err, syncErr) {
		t.Fatalf("connect error = %v, want it to wrap %v", err, syncErr)
	}
	if got := manager.StateOf("inv1"); got != StateOffGrid {
		t.Fatalf("state = %s, want off-grid (must not mark on-grid on failure)", got)
	}
	if le := manager.LastError("inv1"); le == nil || !strings.Contains(le.Error(), syncErr.Error()) {
		t.Fatalf("last error = %v, want it to contain %v", le, syncErr)
	}
	gs, _ := table.GridStateOf("inv1")
	if gs != inverter.GridOffGrid {
		t.Fatalf("table grid state = %s, want off-grid", gs)
	}
	inv, _ := table.Get("inv1")
	if inv.LastGridErr == "" {
		t.Fatal("inverter LastGridErr must record the failure, got empty")
	}
	if inv.GridErrCount == 0 {
		t.Fatal("inverter GridErrCount must be incremented on failure")
	}
}

// TestConnectRetriesBeforeFailing verifies the retry contract: the sync is
// attempted maxSyncRetries times and the failure count reflects that.
func TestConnectRetriesBeforeFailing(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	manager := NewManager(table, failingComm{syncErr: errors.New("down")})

	if err := manager.Connect("inv1"); err == nil {
		t.Fatal("expected failure after retries, got nil")
	}
	if got := manager.HiddenFailures("inv1"); got != maxSyncRetries {
		t.Fatalf("hidden failure count = %d, want %d", got, maxSyncRetries)
	}
}

// TestConnectRecoversAfterFlakySync confirms a transient failure that clears
// within the retry budget still results in a successful connect.
func TestConnectRecoversAfterFlakySync(t *testing.T) {
	table := inverter.NewTable()
	_ = table.Register(&inverter.Inverter{ID: "inv1", Serial: "S1", Model: "PV-100K"})
	comm := &flakyComm{successOn: 2} // fails first, succeeds second
	manager := NewManager(table, comm)

	if err := manager.Connect("inv1"); err != nil {
		t.Fatalf("connect should succeed after retry: %v", err)
	}
	if got := manager.StateOf("inv1"); got != StateOnGrid {
		t.Fatalf("state = %s, want on-grid after recovery", got)
	}
	// A successful connect clears the prior failure counters.
	if got := manager.HiddenFailures("inv1"); got != 0 {
		t.Fatalf("hidden failure count after success = %d, want 0", got)
	}
	inv, _ := table.Get("inv1")
	if inv.LastGridErr != "" {
		t.Fatalf("inverter LastGridErr after success = %q, want empty", inv.LastGridErr)
	}
}
