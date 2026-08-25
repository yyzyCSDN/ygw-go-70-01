package inverter

import (
	"errors"
	"sort"
)

type Model struct {
	ID          string
	Name        string
	RatedPowerW int
	MinPowerW   int
	MaxVoltageV int
	MaxCurrentA int
	Efficiency  int
	PortCount   int
}

type Catalog struct {
	models map[string]Model
	order  []string
}

var ErrModelExists = errors.New("pvcontrol: model already registered")
var ErrModelUnknown = errors.New("pvcontrol: unknown inverter model")

func NewCatalog() *Catalog {
	return &Catalog{models: make(map[string]Model)}
}

func (c *Catalog) Register(m Model) error {
	if _, ok := c.models[m.ID]; ok {
		return ErrModelExists
	}
	if m.ID == "" || m.Name == "" || m.RatedPowerW <= 0 {
		return errors.New("pvcontrol: invalid model definition")
	}
	c.models[m.ID] = m
	c.order = append(c.order, m.ID)
	sort.Strings(c.order)
	return nil
}

func (c *Catalog) Get(id string) (Model, error) {
	m, ok := c.models[id]
	if !ok {
		return Model{}, ErrModelUnknown
	}
	return m, nil
}

func (c *Catalog) Has(id string) bool {
	_, ok := c.models[id]
	return ok
}

func (c *Catalog) List() []Model {
	out := make([]Model, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.models[id])
	}
	return out
}

func DefaultCatalog() *Catalog {
	c := NewCatalog()
	_ = c.Register(Model{ID: "PV-100K", Name: "Phoenix 100", RatedPowerW: 100000, MinPowerW: 5000, MaxVoltageV: 1100, MaxCurrentA: 180, Efficiency: 98, PortCount: 4})
	_ = c.Register(Model{ID: "PV-50K", Name: "Phoenix 50", RatedPowerW: 50000, MinPowerW: 3000, MaxVoltageV: 1000, MaxCurrentA: 100, Efficiency: 97, PortCount: 3})
	_ = c.Register(Model{ID: "PV-200K", Name: "Titan 200", RatedPowerW: 200000, MinPowerW: 10000, MaxVoltageV: 1500, MaxCurrentA: 320, Efficiency: 99, PortCount: 6})
	return c
}
