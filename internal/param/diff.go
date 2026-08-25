package param

import "pvcontrol/internal/inverter"

func Diff(before, after inverter.ParamSet) []string {
	out := make([]string, 0, 5)
	if before.LimitPowerW != after.LimitPowerW {
		out = append(out, "limit_power_w")
	}
	if before.ReactiveLimit != after.ReactiveLimit {
		out = append(out, "reactive_limit")
	}
	if before.RampRate != after.RampRate {
		out = append(out, "ramp_rate")
	}
	if before.VoltRef != after.VoltRef {
		out = append(out, "volt_ref")
	}
	if before.FreqRef != after.FreqRef {
		out = append(out, "freq_ref")
	}
	return out
}
