package alarm

import (
	"time"

	"pvcontrol/internal/inverter"
)

type AlarmEvent struct {
	InverterID string
	Kind       string
	At         int64
}

type Manager struct {
	table       *inverter.Table
	active      map[string]AlarmEvent
	lastRebuild []inverter.Inverter
	now         func() time.Time
}

func NewManager(table *inverter.Table) *Manager {
	return &Manager{
		table:  table,
		active: make(map[string]AlarmEvent),
		now:    time.Now,
	}
}

func (m *Manager) Raise(invID string, kind string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	if err := m.table.SetState(invID, inverter.StateFault); err != nil {
		return err
	}
	m.active[invID] = AlarmEvent{InverterID: invID, Kind: kind, At: m.now().Unix()}
	m.lastRebuild = m.table.Snapshot()
	return nil
}

func (m *Manager) Resolve(invID string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	delete(m.active, invID)
	return nil
}

func (m *Manager) ActiveCount() int {
	return len(m.active)
}

func (m *Manager) Active() []AlarmEvent {
	out := make([]AlarmEvent, 0, len(m.active))
	for _, event := range m.active {
		out = append(out, event)
	}
	return out
}

func (m *Manager) restoreSource() []inverter.Inverter {
	return m.table.Snapshot()
}

func (m *Manager) RebuildAfterRestart() error {
	invs := m.restoreSource()
	m.lastRebuild = invs
	for _, inv := range invs {
		if inv.State != inverter.StateFault {
			delete(m.active, inv.ID)
			continue
		}
		if _, ok := m.active[inv.ID]; ok {
			continue
		}
		m.active[inv.ID] = AlarmEvent{InverterID: inv.ID, Kind: "fault", At: m.now().Unix()}
	}
	return nil
}

func (m *Manager) LastRebuildCount() int {
	return len(m.lastRebuild)
}
