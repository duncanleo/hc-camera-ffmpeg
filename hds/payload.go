package hds

import (
	"bytes"
)

type PayloadRaw struct {
	Header  []byte
	Message []byte
}

type Payload struct {
	Header  map[string]interface{}
	Message interface{}
}

func (pr PayloadRaw) ConvertToPayload() (Payload, error) {
	var p Payload
	var headerBuffer = bytes.NewBuffer(pr.Header)
	headerData, err := DecodeDataFormat(headerBuffer)

	if err != nil {
		return p, err
	}

	p.Header = headerData.(map[string]interface{})

	var messageBuffer = bytes.NewBuffer(pr.Message)
	messageData, err := DecodeDataFormat(messageBuffer)
	p.Message = messageData

	return p, err
}

func (pr PayloadRaw) Assemble() []byte {
	var respPayload []byte
	var encodedHeaderLen = len(pr.Header)

	respPayload = append(respPayload, byte(encodedHeaderLen))
	respPayload = append(respPayload, pr.Header...)
	respPayload = append(respPayload, pr.Message...)

	return respPayload
}

func NewPayloadRaw(decrypted []byte) PayloadRaw {
	var pr PayloadRaw
	var headerLen = int(decrypted[0])
	pr.Header = decrypted[1 : headerLen+1]

	pr.Message = decrypted[1+headerLen:]

	return pr
}

func (p Payload) Encode() ([]byte, error) {
	encodedHeader, err := EncodeDataFormat(p.Header)
	if err != nil {
		return nil, err
	}

	encodedMessage, err := EncodeDataFormat(p.Message)
	if err != nil {
		return nil, err
	}

	var pr = PayloadRaw{
		Header:  encodedHeader,
		Message: encodedMessage,
	}

	return pr.Assemble(), nil
}

func NewPayloadFromRaw(decrypted []byte) (Payload, error) {
	var pr = NewPayloadRaw(decrypted)

	return pr.ConvertToPayload()
}
