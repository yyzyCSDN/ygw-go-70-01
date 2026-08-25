package comm

import (
	"encoding/binary"
	"errors"

	"github.com/cespare/xxhash/v2"

	"pvcontrol/internal/inverter"
)

var ErrEmptyFrame = errors.New("pvcontrol: empty communication frame")
var ErrFrameTooShort = errors.New("pvcontrol: communication frame too short")
var ErrChecksumMismatch = errors.New("pvcontrol: communication frame checksum mismatch")
var ErrDuplicateFrame = errors.New("pvcontrol: duplicate communication frame")

func EncodeFrame(frame *inverter.Frame) []byte {
	kind := []byte(frame.Kind)
	serial := []byte(frame.Serial)
	headerLen := 4 + 2 + 1 + len(kind) + 1 + len(serial)
	raw := make([]byte, 0, headerLen+len(frame.Payload)+8)
	raw = binary.BigEndian.AppendUint32(raw, uint32(len(frame.Payload)))
	raw = binary.BigEndian.AppendUint16(raw, uint16(frame.Seq))
	raw = append(raw, byte(len(kind)))
	raw = append(raw, kind...)
	raw = append(raw, byte(len(serial)))
	raw = append(raw, serial...)
	raw = append(raw, frame.Payload...)
	checksum := xxhash.Sum64(raw)
	raw = binary.BigEndian.AppendUint64(raw, checksum)
	return raw
}

func ParseFrame(raw []byte) (*inverter.Frame, error) {
	if len(raw) == 0 {
		return nil, ErrEmptyFrame
	}
	if len(raw) < 4+2+1+1+8 {
		return nil, ErrFrameTooShort
	}
	payloadLen := int(binary.BigEndian.Uint32(raw[0:4]))
	seq := int(binary.BigEndian.Uint16(raw[4:6]))
	kindLen := int(raw[6])
	offset := 7
	if offset+kindLen > len(raw)-8 {
		return nil, ErrFrameTooShort
	}
	kind := string(raw[offset : offset+kindLen])
	offset += kindLen
	if offset >= len(raw)-8 {
		return nil, ErrFrameTooShort
	}
	serialLen := int(raw[offset])
	offset++
	if offset+serialLen > len(raw)-8 {
		return nil, ErrFrameTooShort
	}
	serial := string(raw[offset : offset+serialLen])
	offset += serialLen
	if offset+payloadLen > len(raw)-8 {
		return nil, ErrFrameTooShort
	}
	payload := raw[offset : offset+payloadLen]
	offset += payloadLen
	checksum := binary.BigEndian.Uint64(raw[offset : offset+8])
	want := xxhash.Sum64(raw[:offset])
	if checksum != want {
		return nil, ErrChecksumMismatch
	}
	return &inverter.Frame{
		Serial:   serial,
		Kind:     inverter.FrameKind(kind),
		Seq:      seq,
		Payload:  payload,
		Checksum: checksum,
	}, nil
}

func encodePower(watts int) []byte {
	return inverter.EncodePower(watts)
}
