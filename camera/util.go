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
	if motion && doorbell {
		return 0x01 & 0x02
	} else if motion {
		return 0x01
	} else {
		return 0x02
	}
}
