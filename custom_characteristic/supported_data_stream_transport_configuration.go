package custom_characteristic

import "github.com/brutella/hc/characteristic"

const TypeSupportedDataStreamTransportConfiguration = "130"

type SupportedDataStreamTransportConfiguration struct {
	*characteristic.Bytes
}

func NewSupportedDataStreamTransportConfiguration() *SupportedDataStreamTransportConfiguration {
	var char = characteristic.NewBytes(TypeSupportedDataStreamTransportConfiguration)

	char.Format = characteristic.FormatTLV8
	char.Perms = []string{characteristic.PermRead}
	char.Description = "Supported Data Stream Transport Configuration"

	char.SetValue([]byte{})

	return &SupportedDataStreamTransportConfiguration{char}
}
