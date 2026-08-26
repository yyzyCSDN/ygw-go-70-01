package param

import (
	"encoding/binary"

	"pvcontrol/internal/inverter"
)

type Sender interface {
	SendParam(invID string, payload []byte) error
}

type Manager struct {
	table         *inverter.Table
	catalog       *Catalog
	sender        Sender
	delivered     map[string]inverter.ParamSet
	appliedModels map[string]string
	planned       map[string]inverter.ParamSet
}

func NewManager(table *inverter.Table, catalog *Catalog, sender Sender) *Manager {
	return &Manager{
		table:         table,
		catalog:       catalog,
		sender:        sender,
		delivered:     make(map[string]inverter.ParamSet),
		appliedModels: make(map[string]string),
		planned:       make(map[string]inverter.ParamSet),
	}
}

func (m *Manager) PlanFor(invID string) (inverter.ParamSet, error) {
	model, err := m.table.ModelOf(invID)
	if err != nil {
		return inverter.ParamSet{}, err
	}
	if last, ok := m.appliedModels[invID]; ok && m.catalog.Has(last) {
		model = last
	}
	set, err := m.catalog.ForModel(model)
	if err != nil {
		return inverter.ParamSet{}, err
	}
	m.planned[invID] = set
	return set, nil
}

func (m *Manager) Deliver(invID string, next inverter.ParamSet) error {
	inv, err := m.table.Get(invID)
	if err != nil {
		return err
	}
	if err := m.catalog.Validate(next, inv.Model); err != nil {
		return err
	}
	payload := encodeParams(next)
	if err := m.sender.SendParam(invID, payload); err != nil {
		return err
	}
	if err := m.table.ApplyParams(invID, next); err != nil {
		return err
	}
	m.delivered[invID] = next
	m.appliedModels[invID] = inv.Model
	return nil
}

func (m *Manager) Delivered(invID string) (inverter.ParamSet, bool) {
	value, ok := m.delivered[invID]
	return value, ok
}

func (m *Manager) AppliedModel(invID string) string {
	return m.appliedModels[invID]
}

func encodeParams(p inverter.ParamSet) []byte {
	raw := make([]byte, 0, 32)
	raw = append(raw, []byte(p.Model)...)
	raw = append(raw, 0)
	raw = binary.BigEndian.AppendUint32(raw, uint32(p.Rev))
	raw = binary.BigEndian.AppendUint32(raw, uint32(p.LimitPowerW))
	raw = binary.BigEndian.AppendUint32(raw, uint32(p.ReactiveLimit))
	raw = binary.BigEndian.AppendUint32(raw, uint32(p.RampRate))
	raw = binary.BigEndian.AppendUint32(raw, uint32(p.VoltRef))
	raw = binary.BigEndian.AppendUint32(raw, uint32(p.FreqRef))
	return raw
}
