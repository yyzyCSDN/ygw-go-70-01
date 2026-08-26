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
