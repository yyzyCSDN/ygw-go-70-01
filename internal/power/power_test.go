package power

import "testing"

func TestComputeTarget(t *testing.T) {
	cases := []struct {
		capacity int
		wm2      int
		want     int
	}{
		{100000, 0, 0},
		{100000, 400, 40000},
		{100000, 1000, 100000},
		{100000, 1200, 100000},
	}
	for _, c := range cases {
		if got := computeTarget(c.capacity, c.wm2); got != c.want {
			t.Fatalf("computeTarget(%d, %d) = %d, want %d", c.capacity, c.wm2, got, c.want)
		}
	}
}

func TestForecastAverage(t *testing.T) {
	forecast := NewForecast(3)
	forecast.Add(700)
	forecast.Add(500)
	forecast.Add(300)
	if got := forecast.Average(); got != 500 {
		t.Fatalf("average = %d, want 500", got)
	}
	forecast.Add(100)
	if got := forecast.Count(); got != 3 {
		t.Fatalf("window count = %d, want 3", got)
	}
}

func TestDistribute(t *testing.T) {
	inverters := []PlantInverter{
		{ID: "a", RatedPowerW: 100000, MinPowerW: 5000},
		{ID: "b", RatedPowerW: 50000, MinPowerW: 3000},
	}
	assignments := Distribute(90000, inverters)
	total := 0
	for _, a := range assignments {
		total += a.TargetW
	}
	if total != 90000 {
		t.Fatalf("distributed total = %d, want 90000", total)
	}
	if len(assignments) != 2 {
		t.Fatalf("assignments = %d, want 2", len(assignments))
	}
}

type fakeDispatcher struct {
	dispatched []int
}

func (f *fakeDispatcher) DispatchTarget(targetW int) error {
	f.dispatched = append(f.dispatched, targetW)
	return nil
}

// TestUpdateIrradianceRefreshesTarget guards against the regression where a
// change in irradiance between two non-zero values (e.g. 900 -> 400) did not
// refresh the power target: the old code recomputed the target from a stale
// "last" irradiance and short-circuited, so the inverter kept limiting to the
// old target and the plant under-produced.
func TestUpdateIrradianceRefreshesTarget(t *testing.T) {
	d := &fakeDispatcher{}
	m := NewManager(350000, d)

	if err := m.UpdateIrradiance(900); err != nil {
		t.Fatalf("UpdateIrradiance(900): %v", err)
	}
	if m.Target() != 315000 {
		t.Fatalf("after 900 target = %d, want 315000", m.Target())
	}

	// Irradiance drops to 400. The target must immediately follow the new
	// irradiance and the new target must be dispatched to the inverters.
	if err := m.UpdateIrradiance(400); err != nil {
		t.Fatalf("UpdateIrradiance(400): %v", err)
	}
	if m.Target() != 140000 {
		t.Fatalf("after 400 target = %d, want 140000", m.Target())
	}
	if len(d.dispatched) != 2 || d.dispatched[1] != 140000 {
		t.Fatalf("dispatched = %v, want final dispatch of 140000", d.dispatched)
	}
}
