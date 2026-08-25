package param

import (
	"errors"
	"sort"

	"pvcontrol/internal/inverter"
)

type Catalog struct {
	sets  map[string]inverter.ParamSet
	order []string
}

var ErrParamUnknownModel = errors.New("pvcontrol: no parameter set for model")
var ErrParamInvalid = errors.New("pvcontrol: parameter set invalid for model")

func NewCatalog() *Catalog {
	return &Catalog{sets: make(map[string]inverter.ParamSet)}
}

func (c *Catalog) Add(model string, p inverter.ParamSet) error {
	if model == "" || p.Model != model {
		return ErrParamInvalid
	}
	if _, ok := c.sets[model]; ok {
		return errors.New("pvcontrol: duplicate parameter set")
	}
	c.sets[model] = p
	c.order = append(c.order, model)
	sort.Strings(c.order)
	return nil
}

func (c *Catalog) ForModel(model string) (inverter.ParamSet, error) {
	p, ok := c.sets[model]
	if !ok {
		return inverter.ParamSet{}, ErrParamUnknownModel
	}
	return p, nil
}

func (c *Catalog) Has(model string) bool {
	_, ok := c.sets[model]
	return ok
}

func (c *Catalog) List() []inverter.ParamSet {
	out := make([]inverter.ParamSet, 0, len(c.order))
	for _, model := range c.order {
		out = append(out, c.sets[model])
	}
	return out
}

func (c *Catalog) Validate(p inverter.ParamSet, model string) error {
	base, ok := c.sets[model]
	if !ok {
		return ErrParamUnknownModel
	}
	if p.LimitPowerW <= 0 {
		return ErrParamInvalid
	}
	if p.LimitPowerW > base.LimitPowerW*2 {
		return ErrParamInvalid
	}
	if p.RampRate < 0 || p.RampRate > 100 {
		return ErrParamInvalid
	}
	if p.ReactiveLimit < 0 {
		return ErrParamInvalid
	}
	return nil
}

func DefaultCatalog(models *inverter.Catalog) *Catalog {
	c := NewCatalog()
	for _, model := range models.List() {
		base := model.RatedPowerW
		_ = c.Add(model.ID, inverter.ParamSet{
			Model:         model.ID,
			Rev:           1,
			LimitPowerW:   base,
			ReactiveLimit: 40,
			RampRate:      30,
			VoltRef:       1000,
			FreqRef:       50,
		})
	}
	return c
}
