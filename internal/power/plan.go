package power

type PlantPlan struct {
	TargetW     int
	Assignments []Assignment
}

func BuildPlan(irradianceWm2, capacityW int, inverters []PlantInverter) PlantPlan {
	target := computeTarget(capacityW, irradianceWm2)
	return PlantPlan{
		TargetW:     target,
		Assignments: Distribute(target, inverters),
	}
}

func PlanByModel(inverters []PlantInverter) map[string]int {
	out := make(map[string]int)
	for _, inv := range inverters {
		out[inv.Model] += inv.RatedPowerW
	}
	return out
}
