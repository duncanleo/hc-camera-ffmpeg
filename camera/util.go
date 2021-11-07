package camera

import (
	"log"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/tlv8"
)

func setTLV8Payload(c *characteristic.Bytes, v interface{}) {
	if tlv8, err := tlv8.Marshal(v); err == nil {
		c.SetValue(tlv8)
	} else {
		log.Fatal(err)
	}
}

func hksvEventTriggerBitmask(motion, doorbell bool) byte {
	var bits = make([]byte, 0)

	if motion {
		bits = append(bits, 0x01)
	}

	if doorbell {
		bits = append(bits, 0x02)
	}

	var result byte

	for i := 0; i < len(bits); i++ {
		result = result | bits[i]
	}

	return result
}
