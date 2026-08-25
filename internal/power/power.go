package power

import "errors"

var ErrIrradianceInvalid = errors.New("pvcontrol: invalid irradiance value")

type Dispatcher interface {
	DispatchTarget(targetW int) error
}

type Manager struct {
	capacityW         int
	irradiance        int
	lastIrrad         int
	pendingIrradiance int
	targetW           int
	dispatch          Dispatcher
}

func NewManager(capacityW int, dispatch Dispatcher) *Manager {
	return &Manager{capacityW: capacityW, dispatch: dispatch}
}

func (m *Manager) UpdateIrradiance(wm2 int) error {
	if wm2 < 0 {
		return ErrIrradianceInvalid
	}
	m.irradiance = wm2
	m.pendingIrradiance = wm2
	return m.refresh()
}

func (m *Manager) Tick() error {
	m.lastIrrad = m.irradiance
	m.pendingIrradiance = 0
	return m.refresh()
}

func (m *Manager) refresh() error {
	irradiance := m.lastIrrad
	if irradiance == 0 && m.pendingIrradiance != 0 {
		irradiance = m.pendingIrradiance
	}
	target := computeTarget(m.capacityW, irradiance)
	if target == m.targetW {
		return nil
	}
	m.targetW = target
	m.lastIrrad = irradiance
	m.pendingIrradiance = 0
	if m.dispatch != nil {
		return m.dispatch.DispatchTarget(target)
	}
	return nil
}

func (m *Manager) Target() int {
	return m.targetW
}

func (m *Manager) Irradiance() int {
	return m.irradiance
}

func computeTarget(capacityW, wm2 int) int {
	if wm2 <= 0 {
		return 0
	}
	if wm2 >= 1000 {
		return capacityW
	}
	return capacityW * wm2 / 1000
}
