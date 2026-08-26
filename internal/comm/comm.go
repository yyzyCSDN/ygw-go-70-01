package comm

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"pvcontrol/internal/dedup"
	"pvcontrol/internal/inverter"
)

type Client struct {
	table      *inverter.Table
	transport  Transport
	dedup      *dedup.Cache
	frameRing  *dedup.Ring
	metrics    *Metrics
	sessions   map[string]*Session
	lastSeqSig map[string]uint64
	leaseLeaks []int
	openLeases []string
	patrolled  []string
	patrolRounds int
	seq        int
	mu         sync.Mutex
}

func NewClient(table *inverter.Table, transport Transport) *Client {
	return &Client{
		table:      table,
		transport:  transport,
		dedup:      dedup.NewCache(),
		frameRing:  dedup.NewRing(1024),
		metrics:    &Metrics{},
		sessions:   make(map[string]*Session),
		lastSeqSig: make(map[string]uint64),
	}
}

func (c *Client) nextSeq() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	return c.seq
}

func (c *Client) serialOf(invID string) string {
	inv, err := c.table.Get(invID)
	if err != nil {
		return ""
	}
	return inv.Serial
}

func (c *Client) OpenSession(invID string) (*Session, error) {
	handle, err := c.transport.Open(invID)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	s := &Session{ID: fmt.Sprintf("%s-%d", invID, c.seq), Inverter: invID, handle: handle}
	c.sessions[invID] = s
	c.mu.Unlock()
	return s, nil
}

func (c *Client) CloseSession(invID string) error {
	c.mu.Lock()
	s, ok := c.sessions[invID]
	if ok {
		delete(c.sessions, invID)
	}
	c.mu.Unlock()
	if ok && s.handle != nil {
		return s.handle.Close()
	}
	return nil
}

func (c *Client) SessionCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.sessions)
}

func (c *Client) SendCommand(invID string, cmd inverter.Command, payload []byte) error {
	inv, err := c.table.Get(invID)
	if err != nil {
		return err
	}
	frame := &inverter.Frame{Serial: inv.Serial, Kind: inverter.FrameKind(cmd), Seq: c.nextSeq(), Payload: payload}
	raw := EncodeFrame(frame)
	s, err := c.OpenSession(invID)
	if err != nil {
		return err
	}
	defer c.CloseSession(invID)
	if err := s.handle.Write(raw); err != nil {
		c.metrics.RecordFailure()
		return err
	}
	reply, err := s.handle.Read()
	if err != nil {
		c.metrics.RecordFailure()
		return err
	}
	ack, err := ParseFrame(reply)
	if err != nil {
		c.metrics.RecordFailure()
		return err
	}
	if ack.Kind != inverter.FrameAck {
		c.metrics.RecordFailure()
		return errors.New("pvcontrol: unexpected device reply")
	}
	c.metrics.RecordSuccess()
	return nil
}

func (c *Client) SyncGrid(invID string) error {
	return c.SendCommand(invID, inverter.CommandSyncGrid, nil)
}

func (c *Client) SendParam(invID string, payload []byte) error {
	return c.SendCommand(invID, inverter.CommandParam, payload)
}

func (c *Client) DispatchTarget(targetW int) error {
	invs := c.table.Snapshot()
	for _, inv := range invs {
		if err := c.SendCommand(inv.ID, inverter.CommandLimit, encodePower(targetW)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) DispatchSequence(invID string, seq SwitchSequence) error {
	sig := c.sequenceSignature(seq)
	c.lastSeqSig[invID] = sig
	payload := encodeSequence(seq)
	return c.SendCommand(invID, inverter.CommandSwitch, payload)
}

func (c *Client) sequenceSignature(seq SwitchSequence) uint64 {
	return c.dedup.Key(seq.InverterID, seq.Target, string(encodeSequence(seq)))
}

func (c *Client) LastSequenceSignature(invID string) uint64 {
	return c.lastSeqSig[invID]
}

func (c *Client) WritebackState(invID string, state inverter.State) error {
	payload := []byte(state)
	return c.SendCommand(invID, inverter.CommandStateWb, payload)
}

func (c *Client) MarkDedup(invID string, cmd inverter.Command, payload []byte) {
	c.dedup.Mark(c.dedupKey(invID, cmd, payload), c.dedupGroup(invID, cmd))
}

func (c *Client) CheckDedup(invID string, cmd inverter.Command, payload []byte) bool {
	return c.dedup.Check(c.dedupKey(invID, cmd, payload))
}

func (c *Client) ClearDedup(invID string, cmd inverter.Command) error {
	c.dedup.ClearGroup(c.dedupGroup(invID, cmd))
	return nil
}

func (c *Client) dedupKey(invID string, cmd inverter.Command, payload []byte) uint64 {
	return c.dedup.Key(invID, string(cmd), string(payload))
}

func (c *Client) dedupGroup(invID string, cmd inverter.Command) string {
	return fmt.Sprintf("%s|%s", invID, cmd)
}

func (c *Client) Metrics() *Metrics {
	return c.metrics
}

func (c *Client) Patrol(invIDs []string) error {
	for _, invID := range invIDs {
		leaseID, err := c.table.AcquireLease(invID)
		if err != nil {
			continue
		}
		c.leaseLeaks = append(c.leaseLeaks, leaseID)
		s, err := c.OpenSession(invID)
		if err != nil {
			continue
		}
		c.openLeases = append(c.openLeases, invID)
		c.patrolled = append(c.patrolled, invID)
		frame := &inverter.Frame{Serial: c.serialOf(invID), Kind: inverter.FrameStatus, Seq: c.nextSeq(), Payload: nil}
		if err := s.handle.Write(EncodeFrame(frame)); err != nil {
			continue
		}
		_, _ = s.handle.Read()
	}
	c.patrolRounds++
	return nil
}

func (c *Client) Enqueue(invID string, seq int, kind string, payload []byte) error {
	return c.table.EnqueueMessage(invID, inverter.QueuedMessage{
		InverterID: invID,
		Seq:        seq,
		Kind:       kind,
		Payload:    payload,
	})
}

func (c *Client) ProcessBatch(batch []inverter.QueuedMessage) ([]inverter.MessageResult, error) {
	results := make([]inverter.MessageResult, 0, len(batch))
	for _, msg := range batch {
		var res inverter.MessageResult
		if err := c.SendCommand(msg.InverterID, inverter.Command(msg.Kind), msg.Payload); err != nil {
			res = inverter.MessageResult{Status: inverter.MessageFailed, ErrText: err.Error(), At: time.Now().Unix()}
			results = append(results, res)
			_ = c.table.WriteMessageResult(msg.InverterID, msg.Seq, res)
			return results, err
		}
		res = inverter.MessageResult{Status: inverter.MessageSent, At: time.Now().Unix()}
		results = append(results, res)
		if err := c.table.WriteMessageResult(msg.InverterID, msg.Seq, res); err != nil {
			return results, err
		}
	}
	return results, nil
}

func (c *Client) Ingest(raw []byte) (*inverter.Frame, error) {
	frame, err := ParseFrame(raw)
	if err != nil {
		return nil, err
	}
	c.metrics.RecordFrame()
	key := c.dedup.Key(frame.Serial, string(frame.Kind), string(frame.Payload))
	if !c.frameRing.CheckAdd(key) {
		return nil, ErrDuplicateFrame
	}
	if err := c.table.ApplyFrame(frame); err != nil {
		return nil, err
	}
	return frame, nil
}

type Hub struct {
	mu      sync.Mutex
	clients map[string]*Client
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func (h *Hub) Add(name string, client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if name == "" || client == nil {
		return errors.New("pvcontrol: invalid hub client")
	}
	if _, ok := h.clients[name]; ok {
		return errors.New("pvcontrol: hub client already registered")
	}
	h.clients[name] = client
	return nil
}

func (h *Hub) Client(name string) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client, ok := h.clients[name]
	if !ok {
		return nil, errors.New("pvcontrol: unknown hub client")
	}
	return client, nil
}

func (h *Hub) ClientNames() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.clients))
	for name := range h.clients {
		out = append(out, name)
	}
	return out
}

func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) IngestWith(name string, raw []byte) (*inverter.Frame, error) {
	client, err := h.Client(name)
	if err != nil {
		return nil, err
	}
	return client.Ingest(raw)
}
