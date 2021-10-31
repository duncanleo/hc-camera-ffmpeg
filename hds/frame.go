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

func NewFrame(r io.Reader) (Frame, error) {
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
