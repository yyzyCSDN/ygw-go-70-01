package inverter

type ParamSet struct {
	Model         string
	Rev           int
	LimitPowerW   int
	ReactiveLimit int
	RampRate      int
	VoltRef       int
	FreqRef       int
}

type MessageStatus string

const (
	MessagePending MessageStatus = "pending"
	MessageSent    MessageStatus = "sent"
	MessageFailed  MessageStatus = "failed"
)

type QueuedMessage struct {
	InverterID string
	Seq        int
	Kind       string
	Payload    []byte
}

type MessageResult struct {
	Status  MessageStatus
	ErrText string
	At      int64
}

type GridState string

const (
	GridOffGrid    GridState = "off-grid"
	GridSyncing    GridState = "syncing"
	GridOnGrid     GridState = "on-grid"
	GridSeparating GridState = "separating"
)

type FrameKind string

const (
	FrameSync     FrameKind = "sync"
	FrameStart    FrameKind = "start"
	FrameStop     FrameKind = "stop"
	FrameParam    FrameKind = "param"
	FrameLimit    FrameKind = "limit"
	FrameStatus   FrameKind = "status"
	FrameAck      FrameKind = "ack"
	FrameRecover  FrameKind = "recover"
	FrameSwitch   FrameKind = "switch"
	FrameStateWb  FrameKind = "state-writeback"
)

type Frame struct {
	Serial   string
	Kind     FrameKind
	Seq      int
	Payload  []byte
	Checksum uint64
}
