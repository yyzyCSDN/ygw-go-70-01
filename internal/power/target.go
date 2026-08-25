package power

import "sort"

type PlantInverter struct {
	ID          string
	Model       string
	RatedPowerW int
	MinPowerW   int
}

type Assignment struct {
	InverterID string
	TargetW    int
}

func Distribute(plantTargetW int, inverters []PlantInverter) []Assignment {
	rated := 0
	for _, inv := range inverters {
		rated += inv.RatedPowerW
	}
	if rated <= 0 {
		return []Assignment{}
	}
	sorted := append([]PlantInverter(nil), inverters...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	out := make([]Assignment, 0, len(sorted))
	remaining := plantTargetW
	active := len(sorted)
	for index, inv := range sorted {
		share := 0
		if index < len(sorted)-1 {
			share = inv.RatedPowerW * plantTargetW / rated
		} else {
			share = remaining
		}
		if share < inv.MinPowerW && remaining >= inv.MinPowerW {
			share = inv.MinPowerW
		}
		if share > inv.RatedPowerW {
			share = inv.RatedPowerW
		}
		if share > remaining {
			share = remaining
		}
		out = append(out, Assignment{InverterID: inv.ID, TargetW: share})
		remaining -= share
		if remaining < 0 {
			remaining = 0
		}
		active--
	}
	return out
}

func PlantCapacity(inverters []PlantInverter) int {
	total := 0
	for _, inv := range inverters {
		total += inv.RatedPowerW
	}
	return total
}
