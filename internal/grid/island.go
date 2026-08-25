package grid

type IslandGuard struct {
	minFreqHz float64
	maxFreqHz float64
}

func NewIslandGuard(minFreqHz, maxFreqHz float64) *IslandGuard {
	return &IslandGuard{minFreqHz: minFreqHz, maxFreqHz: maxFreqHz}
}

func (g *IslandGuard) Trip(frequencyHz float64) bool {
	return frequencyHz < g.minFreqHz || frequencyHz > g.maxFreqHz
}

func (g *IslandGuard) Clear(frequencyHz float64) bool {
	return frequencyHz >= g.minFreqHz && frequencyHz <= g.maxFreqHz
}

func (m *Manager) MarkIsland(invID string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	m.states[invID] = StateOffGrid
	m.lastErr[invID] = ErrSyncFailed
	return m.table.MarkGridResult(invID, ErrSyncFailed)
}

func (m *Manager) RestoreFromIsland(invID string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	m.states[invID] = StateSyncing
	m.states[invID] = StateOnGrid
	m.lastErr[invID] = nil
	return m.table.MarkGridResult(invID, nil)
}
