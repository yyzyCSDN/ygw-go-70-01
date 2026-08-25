package power

import "errors"

var ErrIrradianceInvalid = errors.New("pvcontrol: invalid irradiance value")

type Dispatcher interface {
	DispatchTarget(targetW int) error
}

type Manager struct {
	capacityW  int
	irradiance int
	lastIrrad  int
	targetW    int
	dispatch   Dispatcher
}

func NewManager(capacityW int, dispatch Dispatcher) *Manager {
	return &Manager{capacityW: capacityW, dispatch: dispatch}
}

func (m *Manager) UpdateIrradiance(wm2 int) error {
	if wm2 < 0 {
		return ErrIrradianceInvalid
	}
	m.irradiance = wm2
	return m.refresh()
}

func (m *Manager) Tick() error {
	// Re-evaluate the target on every tick using the latest irradiance so
	// that changes observed since the last refresh are reflected even when
	// the irradiance was set by a path that did not immediately dispatch.
	return m.refresh()
}

func (m *Manager) refresh() error {
	irradiance := m.irradiance
	target := computeTarget(m.capacityW, irradiance)
	if target == m.targetW {
		return nil
	}
	m.targetW = target
	m.lastIrrad = irradiance
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
