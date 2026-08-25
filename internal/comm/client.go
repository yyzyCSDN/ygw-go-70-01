package comm

type Transport interface {
	Open(invID string) (Handle, error)
}

type Handle interface {
	ID() string
	Write(payload []byte) error
	Read() ([]byte, error)
	Close() error
}

type Session struct {
	ID       string
	Inverter string
	handle   Handle
}

func encodeSequence(seq SwitchSequence) []byte {
	raw := make([]byte, 0, 64)
	raw = append(raw, []byte(seq.InverterID)...)
	raw = append(raw, 0)
	raw = append(raw, []byte(seq.Target)...)
	raw = append(raw, 0)
	raw = append(raw, byte(seq.Version))
	for _, step := range seq.Steps {
		raw = append(raw, byte(step.Order))
		raw = append(raw, []byte(step.InverterID)...)
		raw = append(raw, 0)
		raw = append(raw, []byte(step.Action)...)
		raw = append(raw, 0)
	}
	return raw
}
