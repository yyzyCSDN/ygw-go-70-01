package inverter

type PlantStats struct {
	Total    int
	Running  int
	Fault    int
	Stopped  int
	Derating int
	Starting int
	OutputW  int
	Capacity int
}

func CollectStats(table *Table, catalog *Catalog) PlantStats {
	var stats PlantStats
	for _, inv := range table.Snapshot() {
		stats.Total++
		stats.OutputW += inv.OutputW
		if model, err := catalog.Get(inv.Model); err == nil {
			stats.Capacity += model.RatedPowerW
		}
		switch inv.State {
		case StateRunning:
			stats.Running++
		case StateFault:
			stats.Fault++
		case StateStopped:
			stats.Stopped++
		case StateDerating:
			stats.Derating++
		case StateStarting:
			stats.Starting++
		}
	}
	return stats
}

func AvailableCapacity(stats PlantStats) int {
	if stats.Total == 0 {
		return 0
	}
	ratio := stats.Running + stats.Derating
	return stats.Capacity * ratio / stats.Total
}

func OutputRatio(stats PlantStats) int {
	if stats.Capacity == 0 {
		return 0
	}
	return stats.OutputW * 100 / stats.Capacity
}
