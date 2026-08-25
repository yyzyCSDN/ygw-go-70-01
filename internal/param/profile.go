package param

import "pvcontrol/internal/inverter"

type Profile struct {
	Model     string
	Inverters int
	TotalW    int
	AverageW  int
}

func Summarize(table *inverter.Table, catalog *Catalog) []Profile {
	profiles := make(map[string]*Profile)
	for _, inv := range table.Snapshot() {
		p, ok := profiles[inv.Model]
		if !ok {
			p = &Profile{Model: inv.Model}
			profiles[inv.Model] = p
		}
		p.Inverters++
		p.TotalW += inv.Params.LimitPowerW
	}
	keys := make([]string, 0, len(profiles))
	for key := range profiles {
		keys = append(keys, key)
	}
	sortStrings(keys)
	out := make([]Profile, 0, len(keys))
	for _, key := range keys {
		p := profiles[key]
		if p.Inverters > 0 {
			p.AverageW = p.TotalW / p.Inverters
		}
		out = append(out, *p)
	}
	return out
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
