package inverter

import (
	"errors"
	"time"
)

type Inverter struct {
	ID             string
	Serial         string
	Model          string
	State          State
	GridState      GridState
	Params         ParamSet
	ParamRev       int
	ParamAppliedAt time.Time
	Messages       map[int]MessageResult
	Queue          map[int]QueuedMessage
	lastBatch      map[int]MessageResult
	Leases         map[int]bool
	LastGridErr    string
	GridErrCount   int
	ModelRev       int
	ModelChangedAt time.Time
	OutputW        int
}

var ErrNotFound = errors.New("pvcontrol: inverter not found")
var ErrExists = errors.New("pvcontrol: inverter already registered")
var ErrNilFrame = errors.New("pvcontrol: nil communication frame")

type Table struct {
	inverters      map[string]*Inverter
	modelIndex     map[string]string
	restoreState   []Inverter
	leaseSeq       int
	now            func() time.Time
}

func NewTable() *Table {
	return &Table{
		inverters:  make(map[string]*Inverter),
		modelIndex: make(map[string]string),
		now:        time.Now,
	}
}

func (t *Table) Register(inv *Inverter) error {
	if inv == nil || inv.ID == "" || inv.Serial == "" {
		return errors.New("pvcontrol: invalid inverter registration")
	}
	if _, ok := t.inverters[inv.ID]; ok {
		return ErrExists
	}
	if inv.Messages == nil {
		inv.Messages = make(map[int]MessageResult)
	}
	if inv.Queue == nil {
		inv.Queue = make(map[int]QueuedMessage)
	}
	if inv.lastBatch == nil {
		inv.lastBatch = make(map[int]MessageResult)
	}
	if inv.Leases == nil {
		inv.Leases = make(map[int]bool)
	}
	inv.State = StateStopped
	inv.GridState = GridOffGrid
	t.inverters[inv.ID] = inv
	t.modelIndex[inv.ID] = inv.Model
	return nil
}

func (t *Table) Get(id string) (*Inverter, error) {
	inv, ok := t.inverters[id]
	if !ok {
		return nil, ErrNotFound
	}
	return inv, nil
}

func (t *Table) State(id string) (State, error) {
	inv, err := t.Get(id)
	if err != nil {
		return "", err
	}
	return inv.State, nil
}

func (t *Table) SetState(id string, s State) error {
	inv, err := t.Get(id)
	if err != nil {
		return err
	}
	if !CanTransition(inv.State, s) {
		return TransitionError(inv.State, s)
	}
	inv.State = s
	return nil
}

func (t *Table) GridStateOf(id string) (GridState, error) {
	inv, err := t.Get(id)
	if err != nil {
		return "", err
	}
	return inv.GridState, nil
}

func (t *Table) MarkGridResult(id string, err error) error {
	inv, lookupErr := t.Get(id)
	if lookupErr != nil {
		return lookupErr
	}
	if err != nil {
		inv.GridState = GridOffGrid
		inv.LastGridErr = err.Error()
		inv.GridErrCount++
		return nil
	}
	inv.GridState = GridOnGrid
	inv.LastGridErr = ""
	return nil
}

func (t *Table) Params(id string) (ParamSet, error) {
	inv, err := t.Get(id)
	if err != nil {
		return ParamSet{}, err
	}
	return inv.Params, nil
}

func (t *Table) ApplyParams(id string, p ParamSet) error {
	inv, err := t.Get(id)
	if err != nil {
		return err
	}
	inv.Params = p
	inv.ParamRev++
	inv.ParamAppliedAt = t.now()
	return nil
}

func (t *Table) ModelOf(id string) (string, error) {
	if _, ok := t.inverters[id]; !ok {
		return "", ErrNotFound
	}
	model, ok := t.modelIndex[id]
	if !ok {
		return "", ErrModelUnknown
	}
	return model, nil
}

func (t *Table) Replace(id, serial, model string) error {
	inv, err := t.Get(id)
	if err != nil {
		return err
	}
	inv.Serial = serial
	inv.Model = model
	inv.ModelRev++
	inv.ModelChangedAt = t.now()
	t.modelIndex[id] = model
	return nil
}

func (t *Table) liveSnapshot() []Inverter {
	ids := make([]string, 0, len(t.inverters))
	for id := range t.inverters {
		ids = append(ids, id)
	}
	sortStrings(ids)
	out := make([]Inverter, 0, len(ids))
	for _, id := range ids {
		inv := *t.inverters[id]
		out = append(out, inv)
	}
	return out
}

func (t *Table) Snapshot() []Inverter {
	return t.liveSnapshot()
}

func (t *Table) FreezeRestoreState() {
	t.restoreState = t.liveSnapshot()
}

func (t *Table) RestoreState() []Inverter {
	if t.restoreState == nil {
		return t.liveSnapshot()
	}
	return t.restoreState
}

func (t *Table) EnqueueMessage(id string, msg QueuedMessage) error {
	inv, err := t.Get(id)
	if err != nil {
		return err
	}
	inv.Queue[msg.Seq] = msg
	inv.Messages[msg.Seq] = MessageResult{Status: MessagePending, At: t.now().Unix()}
	return nil
}

func (t *Table) PendingMessages(id string) ([]QueuedMessage, error) {
	inv, err := t.Get(id)
	if err != nil {
		return nil, err
	}
	seqs := make([]int, 0, len(inv.Queue))
	for seq := range inv.Queue {
		seqs = append(seqs, seq)
	}
	sortInts(seqs)
	out := make([]QueuedMessage, 0, len(seqs))
	for _, seq := range seqs {
		msg := inv.Queue[seq]
		if res, ok := inv.Messages[seq]; ok && res.Status != MessagePending {
			continue
		}
		out = append(out, msg)
	}
	return out, nil
}

func (t *Table) MessageStatus(id string, seq int) (MessageResult, error) {
	inv, err := t.Get(id)
	if err != nil {
		return MessageResult{}, err
	}
	res, ok := inv.Messages[seq]
	if !ok {
		return MessageResult{Status: MessagePending}, nil
	}
	return res, nil
}

func (t *Table) WriteMessageResult(id string, seq int, res MessageResult) error {
	inv, err := t.Get(id)
	if err != nil {
		return err
	}
	prev, ok := inv.lastBatch[seq]
	if ok && prev.Status == MessagePending {
		inv.Messages[seq] = prev
		inv.lastBatch[seq] = res
		return nil
	}
	inv.Messages[seq] = res
	inv.lastBatch[seq] = res
	return nil
}

func (t *Table) MessageStatusesFor(batch []QueuedMessage) map[int]MessageResult {
	if len(batch) == 0 {
		return map[int]MessageResult{}
	}
	inv, err := t.Get(batch[0].InverterID)
	if err != nil {
		return map[int]MessageResult{}
	}
	out := make(map[int]MessageResult, len(inv.lastBatch))
	for seq, res := range inv.lastBatch {
		out[seq] = res
	}
	return out
}

func (t *Table) AcquireLease(id string) (int, error) {
	inv, err := t.Get(id)
	if err != nil {
		return 0, err
	}
	t.leaseSeq++
	inv.Leases[t.leaseSeq] = true
	return t.leaseSeq, nil
}

func (t *Table) ReleaseLease(id string, leaseID int) error {
	inv, err := t.Get(id)
	if err != nil {
		return err
	}
	delete(inv.Leases, leaseID)
	return nil
}

func (t *Table) ActiveLeases(id string) (int, error) {
	inv, err := t.Get(id)
	if err != nil {
		return 0, err
	}
	return len(inv.Leases), nil
}

func (t *Table) GetBySerial(serial string) (*Inverter, error) {
	for _, inv := range t.inverters {
		if inv.Serial == serial {
			return inv, nil
		}
	}
	return nil, ErrNotFound
}

func (t *Table) ApplyFrame(frame *Frame) error {
	if frame == nil {
		return ErrNilFrame
	}
	inv, err := t.GetBySerial(frame.Serial)
	if err != nil {
		return err
	}
	switch frame.Kind {
	case FrameStart:
		if CanTransition(inv.State, StateStarting) {
			inv.State = StateStarting
		}
	case FrameStop:
		if CanTransition(inv.State, StateStopped) {
			inv.State = StateStopped
		}
	case FrameLimit:
		inv.OutputW = decodePower(frame.Payload)
	case FrameStatus:
		inv.OutputW = decodePower(frame.Payload)
	}
	return nil
}

func decodePower(payload []byte) int {
	if len(payload) < 4 {
		return 0
	}
	value := 0
	for _, b := range payload[:4] {
		value = value<<8 | int(b)
	}
	return value
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func sortInts(values []int) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
