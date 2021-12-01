package custom_service

import (
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_characteristic"
)

const TypeDataStreamTransportManagement = "129"

type DataStreamTransportManagement struct {
	*service.Service

	SetupDataStreamTransport *custom_characteristic.SetupDataStreamTransport

	SupportedDataStreamTransportConfiguration *custom_characteristic.SupportedDataStreamTransportConfiguration
	Version                                   *characteristic.Version
}

func NewDataStreamTransportManagement() *DataStreamTransportManagement {
	var svc = DataStreamTransportManagement{}
	svc.Service = service.New(TypeDataStreamTransportManagement)

	svc.SetupDataStreamTransport = custom_characteristic.NewSetupDataStreamTransport()
	svc.AddCharacteristic(svc.SetupDataStreamTransport.Characteristic)

	svc.SupportedDataStreamTransportConfiguration = custom_characteristic.NewSupportedDataStreamTransportConfiguration()
	svc.AddCharacteristic(svc.SupportedDataStreamTransportConfiguration.Characteristic)

	svc.Version = characteristic.NewVersion()
	svc.Version.SetValue("1.0")
	svc.AddCharacteristic(svc.Version.Characteristic)

	return &svc
}
