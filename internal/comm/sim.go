package comm

import (
	"errors"
	"sync"

	"pvcontrol/internal/inverter"
)

type simDevice struct {
	invID  string
	serial string
	mu     sync.Mutex
	outbox [][]byte
}

type SimTransport struct {
	table   *inverter.Table
	mu      sync.Mutex
	devices map[string]*simDevice
}

func NewSimTransport(table *inverter.Table) *SimTransport {
	return &SimTransport{table: table, devices: make(map[string]*simDevice)}
}

func (t *SimTransport) Open(invID string) (Handle, error) {
	inv, err := t.table.Get(invID)
	if err != nil {
		return nil, err
	}
	t.mu.Lock()
	dev, ok := t.devices[invID]
	if !ok {
		dev = &simDevice{invID: invID, serial: inv.Serial}
		t.devices[invID] = dev
	}
	t.mu.Unlock()
	return &simHandle{dev: dev}, nil
}

type simHandle struct {
	dev *simDevice
}

func (h *simHandle) ID() string {
	return h.dev.invID
}

func (h *simHandle) Write(payload []byte) error {
	frame, err := ParseFrame(payload)
	if err != nil {
		return err
	}
	ack := h.dev.reply(frame)
	h.dev.mu.Lock()
	h.dev.outbox = append(h.dev.outbox, ack)
	h.dev.mu.Unlock()
	return nil
}

func (h *simHandle) Read() ([]byte, error) {
	h.dev.mu.Lock()
	defer h.dev.mu.Unlock()
	if len(h.dev.outbox) == 0 {
		return nil, errors.New("pvcontrol: no device reply")
	}
	out := h.dev.outbox[0]
	h.dev.outbox = h.dev.outbox[1:]
	return out, nil
}

func (h *simHandle) Close() error {
	return nil
}

func (d *simDevice) reply(frame *inverter.Frame) []byte {
	kind := inverter.FrameAck
	if frame.Kind == inverter.FrameStatus {
		kind = inverter.FrameStatus
	}
	ack := &inverter.Frame{Serial: d.serial, Kind: kind, Seq: frame.Seq, Payload: []byte("ok")}
	return EncodeFrame(ack)
}
