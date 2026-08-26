package inverter

import "errors"

type Command string

const (
	CommandStart     Command = "start"
	CommandStop      Command = "stop"
	CommandLimit     Command = "limit"
	CommandParam     Command = "param"
	CommandSwitch    Command = "switch"
	CommandSyncGrid  Command = "sync"
	CommandRecover   Command = "recover"
	CommandStateWb   Command = "state-writeback"
)

type CommPort interface {
	SendCommand(invID string, cmd Command, payload []byte) error
	WritebackState(invID string, state State) error
	CheckDedup(invID string, cmd Command, payload []byte) bool
	MarkDedup(invID string, cmd Command, payload []byte)
	ClearDedup(invID string, cmd Command) error
}

type Station struct {
	table          *Table
	comm           CommPort
	swallowedWrites map[string]int
	recoverySeq     map[string]int
}

func NewStation(table *Table, comm CommPort) *Station {
	return &Station{
		table:           table,
		comm:            comm,
		swallowedWrites: make(map[string]int),
		recoverySeq:     make(map[string]int),
	}
}

func (s *Station) Start(invID string) error {
	inv, err := s.table.Get(invID)
	if err != nil {
		return err
	}
	if err := s.table.SetState(invID, StateStarting); err != nil {
		return err
	}
	if err := s.comm.SendCommand(invID, CommandStart, []byte(inv.Serial)); err != nil {
		return err
	}
	return s.table.SetState(invID, StateRunning)
}

func (s *Station) Stop(invID string) error {
	inv, err := s.table.Get(invID)
	if err != nil {
		return err
	}
	if err := s.table.SetState(invID, StateStopped); err != nil {
		return err
	}
	return s.comm.SendCommand(invID, CommandStop, []byte(inv.Serial))
}

func (s *Station) LimitPower(invID string, watts int) error {
	if err := s.table.SetState(invID, StateDerating); err != nil {
		return err
	}
	if err := s.comm.SendCommand(invID, CommandLimit, EncodePower(watts)); err != nil {
		return err
	}
	return s.table.SetState(invID, StateRunning)
}

func (s *Station) Recover(invID string) error {
	inv, err := s.table.Get(invID)
	if err != nil {
		return err
	}
	if inv.State != StateFault {
		return errors.New("pvcontrol: inverter is not in fault state")
	}
	if err := s.table.SetState(invID, StateStopped); err != nil {
		return err
	}
	if err := s.table.SetState(invID, StateStarting); err != nil {
		return err
	}
	if err := s.comm.SendCommand(invID, CommandStart, []byte(inv.Serial)); err != nil {
		return err
	}
	if err := s.comm.WritebackState(invID, StateRunning); err != nil {
		_ = err
		s.swallowedWrites[invID]++
	}
	return s.finalizeRecovery(invID)
}

func (s *Station) rememberLimitDedup(invID string) {
	s.comm.MarkDedup(invID, CommandLimit, nil)
}

func (s *Station) finalizeRecovery(invID string) error {
	s.recoverySeq[invID]++
	s.rememberLimitDedup(invID)
	return s.table.SetState(invID, StateRunning)
}

func EncodePower(watts int) []byte {
	return []byte{
		byte(watts >> 24),
		byte(watts >> 16),
		byte(watts >> 8),
		byte(watts),
	}
}
