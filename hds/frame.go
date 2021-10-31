package hds

import (
	"encoding/binary"
	"io"
)

type Frame struct {
	Header  [4]byte
	Payload []byte
	AuthTag [16]byte
}

func (f Frame) Assemble() []byte {
	var frame []byte

	frame = append(frame, f.Header[:]...)
	frame = append(frame, f.Payload...)
	frame = append(frame, f.AuthTag[:]...)

	return frame
}

func NewFrameFromReader(r io.Reader) (Frame, error) {
	var p Frame

	var err error

	_, err = r.Read(p.Header[:])
	if err != nil {
		return p, err
	}

	var lengthBytes = p.Header[1:]
	lengthBytes = append([]byte{0}, lengthBytes...)

	var payloadLength = binary.BigEndian.Uint32(lengthBytes)
	p.Payload = make([]byte, payloadLength)

	_, err = r.Read(p.Payload)
	if err != nil {
		return p, err
	}

	_, err = r.Read(p.AuthTag[:])
	return p, err
}

func NewFrameFromPayloadAndSession(payload []byte, sess *HDSSession) (Frame, error) {
	var f Frame
	var payloadLen = len(payload)

	var header = make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(payloadLen))
	header[0] = 0x01

	encrypted, mac, err := sess.Encrypt(payload, header)
	if err != nil {
		return f, err
	}

	var headerFixed [4]byte
	copy(headerFixed[:], header)

	f.Header = headerFixed
	f.Payload = encrypted
	f.AuthTag = mac

	return f, nil
}
