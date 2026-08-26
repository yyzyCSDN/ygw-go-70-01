package comm

type SwitchStep struct {
	Order      int
	InverterID string
	Action     string
}

type SwitchSequence struct {
	InverterID string
	Target     string
	Version    int
	Steps      []SwitchStep
}

func (s SwitchSequence) StepCount() int {
	return len(s.Steps)
}

func (s SwitchSequence) HasStep(invID string) bool {
	for _, step := range s.Steps {
		if step.InverterID == invID {
			return true
		}
	}
	return false
}
