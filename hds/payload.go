package hds

import "bytes"

type Payload struct {
	Header  []byte
	Message []byte
}

func NewPayload(decrypted []byte) (Payload, error) {
	var p Payload

	var headerLen = int(decrypted[0])
	p.Header = decrypted[1 : headerLen+1]
	p.Message = decrypted[1+headerLen:]

	return p, nil
}

func (p Payload) DecodedHeader() (interface{}, error) {
	var buf = bytes.NewBuffer(p.Header)

	return DecodeDataFormat(buf)
}

func (p Payload) DecodedMessage() (interface{}, error) {
	var buf = bytes.NewBuffer(p.Message)

	return DecodeDataFormat(buf)
}
