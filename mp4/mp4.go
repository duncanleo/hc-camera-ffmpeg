package mp4

import (
	"bytes"
	"encoding/binary"
)

type Chunk struct {
	Size     uint32
	MainType string
	SubType  string
	Data     []byte
}

func (c Chunk) Assemble() ([]byte, error) {
	var buf bytes.Buffer
	var err error

	var size = make([]byte, 4)
	binary.BigEndian.PutUint32(size, c.Size)
	_, err = buf.Write(size)
	if err != nil {
		return nil, err
	}

	_, err = buf.WriteString(c.MainType)
	if err != nil {
		return nil, err
	}

	if len(c.SubType) > 0 {
		_, err = buf.WriteString(c.SubType)
		if err != nil {
			return nil, err
		}
	}

	_, err = buf.Write(c.Data)

	return buf.Bytes(), err
}
