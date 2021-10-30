package camera

import (
	"log"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/tlv8"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_service"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
)

func createDataStreamService() *custom_service.DataStreamTransportManagement {
	var svc = custom_service.NewDataStreamTransportManagement()

	setTLV8Payload(
		svc.SupportedDataStreamTransportConfiguration.Bytes,
		hsv.SupportedDataStreamTransportConfiguration{
			TransferTransportConfigurations: []hsv.TransferTransportConfiguration{
				{
					TransportType: 0,
				},
			},
		})

	svc.SetupDataStreamTransport.OnValueRemoteUpdate(func(value []byte) {

		var request hsv.SetupDataStreamSessionRequest
		var err error

		err = tlv8.Unmarshal(value, &request)

		if err != nil {
			log.Printf("SetupDataStreamTransport onValueRemoteUpdate Error Decodin=%s\n", err)
			return
		}

		log.Printf("SetupDataStreamTransport onValueRemoteUpdate %+v\n", request)
	})

	svc.SetupDataStreamTransport.OnValueUpdate(func(c *characteristic.Characteristic, newValue, oldValue interface{}) {
		log.Printf("SetupDataStreamTransport OnValueUpdate %+v %+v\n", newValue, oldValue)
	})

	return svc
}
