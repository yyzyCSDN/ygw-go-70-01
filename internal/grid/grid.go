package grid

import (
	"errors"
	"fmt"
	"sort"

	"pvcontrol/internal/comm"
	"pvcontrol/internal/inverter"
)

type GridState string

const (
	StateOffGrid    GridState = "off-grid"
	StateSyncing    GridState = "syncing"
	StateOnGrid     GridState = "on-grid"
	StateSeparating GridState = "separating"
)

var ErrSyncFailed = errors.New("pvcontrol: grid sync failed")
var ErrNoSequence = errors.New("pvcontrol: no switch sequence for inverter")

type CommPort interface {
	SyncGrid(invID string) error
	DispatchSequence(invID string, seq comm.SwitchSequence) error
}

type Manager struct {
	table          *inverter.Table
	comm           CommPort
	states         map[string]GridState
	members        map[string]bool
	sequences      map[string]comm.SwitchSequence
	lastDispatched map[string]comm.SwitchSequence
	lastErr        map[string]error
	version        int
}

func NewManager(table *inverter.Table, port CommPort) *Manager {
	return &Manager{
		table:          table,
		comm:           port,
		states:         make(map[string]GridState),
		members:        make(map[string]bool),
		sequences:      make(map[string]comm.SwitchSequence),
		lastDispatched: make(map[string]comm.SwitchSequence),
		lastErr:        make(map[string]error),
	}
}

func (m *Manager) StateOf(invID string) GridState {
	if state, ok := m.states[invID]; ok {
		return state
	}
	return StateOffGrid
}

func (m *Manager) LastError(invID string) error {
	return m.lastErr[invID]
}

func (m *Manager) SequenceOf(invID string) (comm.SwitchSequence, error) {
	seq, ok := m.sequences[invID]
	if !ok {
		return comm.SwitchSequence{}, ErrNoSequence
	}
	return seq, nil
}

func (m *Manager) markConnected(invID string) {
	m.states[invID] = StateOnGrid
	m.lastErr[invID] = nil
	_ = m.table.MarkGridResult(invID, nil)
}

func (m *Manager) markSyncFailed(invID string, err error) error {
	m.states[invID] = StateOffGrid
	m.lastErr[invID] = err
	_ = m.table.MarkGridResult(invID, err)
	return fmt.Errorf("%w: %v", ErrSyncFailed, err)
}

func (m *Manager) Connect(invID string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	m.states[invID] = StateSyncing
	if err := m.comm.SyncGrid(invID); err != nil {
		return m.markSyncFailed(invID, err)
	}
	m.markConnected(invID)
	return nil
}

func (m *Manager) Disconnect(invID string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	m.states[invID] = StateSeparating
	m.states[invID] = StateOffGrid
	_ = m.table.MarkGridResult(invID, errors.New("pvcontrol: manual grid separation"))
	return nil
}

func (m *Manager) nextVersion() int {
	m.version++
	return m.version
}

func (m *Manager) sortedMembers() []string {
	ids := make([]string, 0, len(m.members))
	for id := range m.members {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (m *Manager) buildSequence(invID string, target GridState) comm.SwitchSequence {
	ids := m.sortedMembers()
	steps := make([]comm.SwitchStep, 0, len(ids))
	for index, id := range ids {
		steps = append(steps, comm.SwitchStep{
			Order:      index,
			InverterID: id,
			Action:     string(target),
		})
	}
	return comm.SwitchSequence{
		InverterID: invID,
		Target:     string(target),
		Version:    m.nextVersion(),
		Steps:      steps,
	}
}

func (m *Manager) SwitchTo(invID string, target GridState) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	m.states[invID] = target
	m.sequences[invID] = m.buildSequence(invID, target)
	return nil
}

func (m *Manager) Resync(invID string) error {
	seq, ok := m.sequences[invID]
	if !ok {
		return ErrNoSequence
	}
	if err := m.comm.DispatchSequence(invID, seq); err != nil {
		return err
	}
	m.lastDispatched[invID] = seq
	return nil
}

func (m *Manager) Join(invID string) error {
	if _, err := m.table.Get(invID); err != nil {
		return err
	}
	m.members[invID] = true
	target := m.StateOf(invID)
	if target == StateOffGrid {
		target = StateOnGrid
	}
	m.states[invID] = target
	for _, id := range m.sortedMembers() {
		m.sequences[id] = m.buildSequence(id, m.StateOf(id))
	}
	return nil
}

func (m *Manager) Rebuild() error {
	for _, id := range m.sortedMembers() {
		seq := m.buildSequence(id, m.StateOf(id))
		m.sequences[id] = seq
		if err := m.comm.DispatchSequence(id, seq); err != nil {
			return err
		}
		m.lastDispatched[id] = seq
	}
	return nil
}

func (m *Manager) MemberCount() int {
	return len(m.members)
}

func (m *Manager) Members() []string {
	return m.sortedMembers()
}
