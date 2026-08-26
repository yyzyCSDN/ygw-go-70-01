package power

type Forecast struct {
	window []int
	limit  int
}

func NewForecast(limit int) *Forecast {
	if limit < 1 {
		limit = 1
	}
	return &Forecast{limit: limit}
}

func (f *Forecast) Add(wm2 int) int {
	f.window = append(f.window, wm2)
	if len(f.window) > f.limit {
		f.window = f.window[len(f.window)-f.limit:]
	}
	return f.Average()
}

func (f *Forecast) Average() int {
	if len(f.window) == 0 {
		return 0
	}
	total := 0
	for _, value := range f.window {
		total += value
	}
	return total / len(f.window)
}

func (f *Forecast) Latest() int {
	if len(f.window) == 0 {
		return 0
	}
	return f.window[len(f.window)-1]
}

func (f *Forecast) Count() int {
	return len(f.window)
}
