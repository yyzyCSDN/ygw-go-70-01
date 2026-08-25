package comm

import "sync"

type Metrics struct {
	mu      sync.Mutex
	Sent    int
	Failed  int
	Retries int
	Frames  int
}

func (m *Metrics) RecordSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent++
	m.Frames++
}

func (m *Metrics) RecordFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Failed++
}

func (m *Metrics) RecordFrame() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Frames++
}

func (m *Metrics) Snapshot() (sent, failed, retries, frames int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Sent, m.Failed, m.Retries, m.Frames
}
