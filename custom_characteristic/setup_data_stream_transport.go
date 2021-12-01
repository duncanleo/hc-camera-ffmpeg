package custom_characteristic

import "github.com/brutella/hc/characteristic"

const TypeSetupDataStreamTransport = "131"

type SetupDataStreamTransport struct {
	*characteristic.Bytes
}

func NewSetupDataStreamTransport() *SetupDataStreamTransport {
	var char = characteristic.NewBytes(TypeSetupDataStreamTransport)

	char.Format = characteristic.FormatTLV8
	char.Perms = []string{characteristic.PermRead, characteristic.PermWrite, PermWriteResponse}
	char.Description = "Setup Data Stream Transport"

	char.SetValue([]byte{})

	return &SetupDataStreamTransport{char}
}
