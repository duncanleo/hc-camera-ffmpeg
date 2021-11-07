package hds

import (
	"bytes"
	"encoding/binary"
	"log"
)

const (
	BooleanTrue  = 0x01
	BooleanFalse = 0x02

	IntegerNegativeOne = 0x07
	IntegerStart       = 0x08
	IntegerEnd         = 0x2E
	IntegerThirtyNine  = 0x02F

	SignedInt16LE = 0x31
	SignedInt32LE = 0x32
	SignedInt64LE = 0x33

	Float32LE = 0x35
	Float64LE = 0x36

	UTF8Start = 0x40
	UTF8End   = 0x6F

	DataStart = 0x70
	DataEnd   = 0x90

	DataSmall      = 0x91
	DataMedium     = 0x92
	DataLarge      = 0x93
	DataExtraLarge = 0x94

	ArrayStart = 0xD0
	ArrayEnd   = 0xDF

	DictionaryStart = 0xE0
	DictionaryEnd   = 0xEE
)

func DecodeDataFormat(r *bytes.Buffer) (interface{}, error) {
	tag, err := r.ReadByte()

	if err != nil {
		return nil, err
	}

	if tag == BooleanTrue {
		return true, nil
	} else if tag == BooleanFalse {
		return false, nil
	} else if tag == IntegerNegativeOne {
		return -1, nil
	} else if tag >= IntegerStart && tag <= IntegerEnd {
		var value = tag - IntegerStart

		return int(value), nil
	} else if tag == IntegerThirtyNine {
		return 39, nil
	} else if tag == SignedInt16LE {
		var data = make([]byte, 2)
		var result int16

		_, err = r.Read(data)
		if err != nil {
			return nil, err
		}

		var buf = bytes.NewBuffer(data)
		err = binary.Read(buf, binary.LittleEndian, &result)

		return result, err
	} else if tag == SignedInt32LE {
		var data = make([]byte, 4)
		var result int32

		_, err = r.Read(data)
		if err != nil {
			return nil, err
		}

		var buf = bytes.NewBuffer(data)
		err = binary.Read(buf, binary.LittleEndian, &result)

		return result, err
	} else if tag == SignedInt64LE {
		var data = make([]byte, 8)
		var result int64

		_, err = r.Read(data)
		if err != nil {
			return nil, err
		}

		var buf = bytes.NewBuffer(data)
		err = binary.Read(buf, binary.LittleEndian, &result)

		return result, err

	} else if tag == Float32LE {
		var data = make([]byte, 4)
		var result float32

		_, err = r.Read(data)
		if err != nil {
			return nil, err
		}

		var buf = bytes.NewBuffer(data)
		err = binary.Read(buf, binary.LittleEndian, &result)

		return result, err
	} else if tag == Float64LE {
		var data = make([]byte, 8)
		var result float64

		_, err = r.Read(data)
		if err != nil {
			return nil, err
		}

		var buf = bytes.NewBuffer(data)
		err = binary.Read(buf, binary.LittleEndian, &result)

		return result, err
	} else if tag >= UTF8Start && tag <= UTF8End {
		var length = int(tag - UTF8Start)

		var result []byte

		for i := 0; i < length; i++ {
			tempByte, err := r.ReadByte()
			if err != nil {
				return nil, err
			}

			result = append(result, tempByte)
		}

		return string(result), err
	} else if tag >= DataStart && tag <= DataEnd {
		var length = int(tag - DataStart)

		var result []byte

		for i := 0; i < length; i++ {
			tempByte, err := r.ReadByte()

			if err != nil {
				return nil, err
			}

			result = append(result, tempByte)
		}

		return result, nil
	} else if tag == DataSmall {
		length, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		var result = make([]byte, length)

		_, err = r.Read(result)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else if tag == DataMedium {
		var lengthBytes = make([]byte, 2)

		_, err = r.Read(lengthBytes)
		if err != nil {
			return nil, err
		}

		var length = binary.LittleEndian.Uint16(lengthBytes)

		var result = make([]byte, length)

		_, err = r.Read(result)
		if err != nil {
			return nil, err
		}

		return result, nil

	} else if tag == DataLarge {
		var lengthBytes = make([]byte, 4)

		_, err = r.Read(lengthBytes)
		if err != nil {
			return nil, err
		}

		var length = binary.LittleEndian.Uint32(lengthBytes)

		var result = make([]byte, length)

		_, err = r.Read(result)
		if err != nil {
			return nil, err
		}

		return result, nil
	} else if tag == DataExtraLarge {
		var lengthBytes = make([]byte, 8)

		_, err = r.Read(lengthBytes)
		if err != nil {
			return nil, err
		}

		var length = binary.LittleEndian.Uint64(lengthBytes)

		var result = make([]byte, length)

		_, err = r.Read(result)
		if err != nil {
			return nil, err
		}

		return result, nil

	} else if tag >= ArrayStart && tag <= ArrayEnd {
		var length = int(tag - ArrayStart)

		var result []interface{}

		for i := 0; i < length; i++ {
			value, err := DecodeDataFormat(r)
			if err != nil {
				return nil, err
			}

			result = append(result, value)
		}

		return result, nil
	} else if tag >= DictionaryStart && tag <= DictionaryEnd {
		var length = int(tag - DictionaryStart)
		var result = make(map[string]interface{})

		for i := 0; i < length; i++ {
			key, err := DecodeDataFormat(r)

			if err != nil {
				return nil, err
			}

			value, err := DecodeDataFormat(r)
			if err != nil {
				return nil, err
			}

			result[key.(string)] = value
		}

		return result, nil
	}

	log.Println("Unhandled byte", tag)

	return nil, nil
}

func EncodeDataFormat(rawData interface{}) ([]byte, error) {
	var result = bytes.NewBuffer(make([]byte, 0))

	switch rawData.(type) {
	case bool:
		var data = rawData.(bool)
		if data {
			result.WriteByte(BooleanTrue)
		} else {
			result.WriteByte(BooleanFalse)
		}
	case string:
		var data = rawData.(string)
		var length = len(data)

		result.WriteByte(byte(UTF8Start + length))
		result.WriteString(data)
	case int:
		var data = rawData.(int)

		switch data {
		case -1:
			result.WriteByte(IntegerNegativeOne)
		case 39:
			result.WriteByte(IntegerThirtyNine)
		default:
			result.WriteByte(byte(data + IntegerStart))
		}
	case int16:
		var data = rawData.(int16)

		var w = bytes.NewBuffer(make([]byte, 0))
		var err = binary.Write(w, binary.LittleEndian, data)

		if err != nil {
			return nil, err
		}

		result.WriteByte(SignedInt16LE)
		result.Write(w.Bytes())
	case int32:
		var data = rawData.(int32)

		var w = bytes.NewBuffer(make([]byte, 0))
		var err = binary.Write(w, binary.LittleEndian, data)

		if err != nil {
			return nil, err
		}

		result.WriteByte(SignedInt32LE)
		result.Write(w.Bytes())
	case int64:
		var data = rawData.(int64)

		var w = bytes.NewBuffer(make([]byte, 0))
		var err = binary.Write(w, binary.LittleEndian, data)

		if err != nil {
			return nil, err
		}

		result.WriteByte(SignedInt64LE)
		result.Write(w.Bytes())
	case []byte:
		var data = rawData.([]byte)

		var size = int64(len(data))

		if size > 255 {
			// uint16
			var lengthBytes = make([]byte, 2)

			binary.LittleEndian.PutUint16(lengthBytes, uint16(size))

			result.WriteByte(DataMedium)
			result.Write(lengthBytes)
			result.Write(data)
		} else if size > 65535 {
			// uint32
			var lengthBytes = make([]byte, 4)

			binary.LittleEndian.PutUint32(lengthBytes, uint32(size))

			result.WriteByte(DataMedium)
			result.Write(lengthBytes)
			result.Write(data)
		} else if size > int64(4294967295) {
			// uint64
			var lengthBytes = make([]byte, 2)

			binary.LittleEndian.PutUint64(lengthBytes, uint64(size))

			result.WriteByte(DataMedium)
			result.Write(lengthBytes)
			result.Write(data)
		} else {
			// uint8

			result.WriteByte(DataSmall)
			result.WriteByte(byte(size))
			result.Write(data)
		}

	case []interface{}:
		var data = rawData.([]interface{})
		var size = len(data)

		result.WriteByte(byte(ArrayStart + size))

		for _, item := range data {
			itemBytes, err := EncodeDataFormat(item)

			if err != nil {
				return nil, err
			}

			result.Write(itemBytes)
		}

	case map[string]interface{}:
		var data = rawData.(map[string]interface{})

		var size = len(data)

		result.WriteByte(byte(DictionaryStart + size))

		for k, v := range data {
			key, err := EncodeDataFormat(k)
			if err != nil {
				return nil, err
			}

			value, err := EncodeDataFormat(v)
			if err != nil {
				return nil, err
			}

			result.Write(key)
			result.Write(value)
		}

	default:
		log.Println("Unsupported type for", rawData)
	}

	return result.Bytes(), nil
}
