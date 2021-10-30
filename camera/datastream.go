package camera

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"log"
	"net"

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

	svc.SetupDataStreamTransport.OnValueUpdateFromConn(func(conn net.Conn, c *characteristic.Characteristic, newValue, oldValue interface{}) {
		var err error

		var newValueString = newValue.(string)
		var newValueStringDecoded []byte
		newValueStringDecoded, err = base64.StdEncoding.DecodeString(newValueString)

		var request hsv.SetupDataStreamSessionRequest

		err = tlv8.Unmarshal(newValueStringDecoded, &request)
		if err != nil {
			log.Println("SetupDataStreamTransport OnValueUpdateFromConn Error Decoding", err)
			return
		}

		log.Printf("SetupDataStreamTransport OnValueUpdateFromConn remoteIP=%+v request=%+v old=%+v\n", conn.RemoteAddr(), request, oldValue)

		var listener net.Listener
		listener, err = net.Listen("tcp", ":0")

		if err != nil {
			log.Println(err)
			return
		}

		var listenerAddress = listener.Addr().(*net.TCPAddr)

		go func() {
			defer listener.Close()
			defer func() {
				log.Println("Ended TCP listener at", listenerAddress)
			}()

			log.Println("Started TCP listener at", listenerAddress)

			tcpConn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				return
			}

			defer tcpConn.Close()

			for {
				var count int
				var err error

				var header = make([]byte, 4)

				count, err = tcpConn.Read(header)
				if err != nil {
					log.Println(err)
					return
				}

				log.Println("[TCP] Received header", count, header)

				var lengthBytes = header[1:]
				lengthBytes = append([]byte{0}, lengthBytes...)

				var payloadLength = binary.BigEndian.Uint32(lengthBytes)
				var payload = make([]byte, payloadLength)

				count, err = tcpConn.Read(payload)
				if err != nil {
					log.Println(err)
					return
				}

				log.Println("[TCP] Received payload", count, payload)

				var authTag = make([]byte, 16)

				count, err = tcpConn.Read(payload)
				if err != nil {
					log.Println(err)
					return
				}

				log.Println("[TCP] Received authTag", count, authTag)

			}

		}()

		var accessoryKeySalt = make([]byte, 32)
		rand.Read(accessoryKeySalt)

		var response = hsv.SetupDataStreamSessionResponse{
			Status: hsv.SetupDataStreamTransportStatusSuccess,
			TransportTypeSessionParameters: hsv.SetupDataStreamTransportConfiguration{
				TCPListeningPort: uint16(listenerAddress.Port),
			},
			AccessoryKeySalt: accessoryKeySalt,
		}

		log.Printf("Writing response %+v\n", response)

		encoded, err := tlv8.Marshal(response)
		if err != nil {
			log.Println(err)
			return
		}

		c.WriteResponse = encoded

	})

	return svc
}
