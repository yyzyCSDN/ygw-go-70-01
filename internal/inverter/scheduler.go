package inverter

import (
	"errors"
	"sync"
)

type Scheduler struct {
	table *Table
	comm  CommPort
	mu    sync.Mutex
	order []string
}

func NewScheduler(table *Table, comm CommPort) *Scheduler {
	return &Scheduler{table: table, comm: comm}
}

func (s *Scheduler) SetOrder(order []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range order {
		if _, err := s.table.Get(id); err != nil {
			return err
		}
	}
	s.order = append([]string(nil), order...)
	return nil
}

func (s *Scheduler) Order() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.order...)
}

func (s *Scheduler) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.order)
}

func (s *Scheduler) StartAll() error {
	order := s.Order()
	if len(order) == 0 {
		return errors.New("pvcontrol: empty start order")
	}
	for _, id := range order {
		inv, err := s.table.Get(id)
		if err != nil {
			return err
		}
		if err := s.table.SetState(id, StateStarting); err != nil {
			return err
		}
		if err := s.comm.SendCommand(id, CommandStart, []byte(inv.Serial)); err != nil {
			return err
		}
		if err := s.table.SetState(id, StateRunning); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) StopAll() error {
	order := s.Order()
	for _, id := range order {
		inv, err := s.table.Get(id)
		if err != nil {
			return err
		}
		if err := s.table.SetState(id, StateStopped); err != nil {
			return err
		}
		if err := s.comm.SendCommand(id, CommandStop, []byte(inv.Serial)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) StartOne(id string) error {
	inv, err := s.table.Get(id)
	if err != nil {
		return err
	}
	if err := s.table.SetState(id, StateStarting); err != nil {
		return err
	}
	if err := s.comm.SendCommand(id, CommandStart, []byte(inv.Serial)); err != nil {
		return err
	}
	return s.table.SetState(id, StateRunning)
}
