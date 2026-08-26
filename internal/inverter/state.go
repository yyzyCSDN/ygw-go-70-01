package inverter

import "fmt"

type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateDerating State = "derating"
	StateFault    State = "fault"
)

var allowedFrom = map[State][]State{
	StateStopped:  {StateStarting, StateFault},
	StateStarting: {StateRunning, StateFault, StateStopped},
	StateRunning:  {StateDerating, StateFault, StateStopped},
	StateDerating: {StateRunning, StateFault, StateStopped},
	StateFault:    {StateStopped},
}

func ValidState(s State) bool {
	switch s {
	case StateStopped, StateStarting, StateRunning, StateDerating, StateFault:
		return true
	}
	return false
}

func CanTransition(from, to State) bool {
	if !ValidState(from) || !ValidState(to) {
		return false
	}
	for _, next := range allowedFrom[from] {
		if next == to {
			return true
		}
	}
	return false
}

func TransitionError(from, to State) error {
	return fmt.Errorf("pvcontrol: illegal inverter transition %s -> %s", from, to)
}
