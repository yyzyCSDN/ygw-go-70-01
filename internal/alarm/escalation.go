package alarm

type Escalator struct {
	levels []string
}

func NewEscalator(levels []string) *Escalator {
	if len(levels) == 0 {
		levels = []string{"info", "warning", "critical"}
	}
	return &Escalator{levels: levels}
}

func (e *Escalator) Level(index int) string {
	if index < 0 {
		index = 0
	}
	if index >= len(e.levels) {
		return e.levels[len(e.levels)-1]
	}
	return e.levels[index]
}

func (e *Escalator) Count() int {
	return len(e.levels)
}

func SeverityIndex(activeCount, perLevel int) int {
	if perLevel <= 0 {
		perLevel = 1
	}
	return activeCount / perLevel
}
