package power

type RampLimiter struct {
	maxStepW int
	lastW    int
}

func NewRampLimiter(maxStepW int) *RampLimiter {
	return &RampLimiter{maxStepW: maxStepW}
}

func (r *RampLimiter) Step(targetW int) int {
	if targetW > r.lastW {
		if targetW-r.lastW > r.maxStepW {
			targetW = r.lastW + r.maxStepW
		}
	} else if r.lastW-targetW > r.maxStepW {
		targetW = r.lastW - r.maxStepW
	}
	if targetW < 0 {
		targetW = 0
	}
	r.lastW = targetW
	return targetW
}

func (r *RampLimiter) Reset(value int) {
	r.lastW = value
}

func (r *RampLimiter) Current() int {
	return r.lastW
}

func DerateTarget(ratedPowerW, percent int) int {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return ratedPowerW * percent / 100
}
