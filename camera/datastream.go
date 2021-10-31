package camera

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"log"
	"net"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/hap"
	"github.com/brutella/hc/tlv8"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_service"
	"github.com/duncanleo/hc-camera-ffmpeg/hds"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
)

var (
	HAPContext *hap.Context
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

		var context = *HAPContext
		var session = context.GetSessionForConnection(conn)

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

		var accessoryKeySalt = make([]byte, 32)
		rand.Read(accessoryKeySalt)

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

				count, err = tcpConn.Read(authTag)
				if err != nil {
					log.Println(err)
					return
				}

				log.Println("[TCP] Received authTag", count, authTag)

				var pairVerifyHandler = session.PairVerifyHandler()
				var sharedKey = pairVerifyHandler.SharedKey()

				var mac [16]byte
				copy(mac[:], authTag)

				sess, err := hds.NewHDSSession(request.ControllerKeySalt, accessoryKeySalt, sharedKey)
				if err != nil {
					log.Println(err)
					return
				}
				decrypted, err := sess.Decrypt(payload, mac, header)
				if err != nil {
					log.Println(err)
					return
				}

				log.Println("DECRYPTION", decrypted, err, len(decrypted))

				var headerLen = int(decrypted[0])
				var payloadHeader = decrypted[1 : headerLen+1]
				var payloadMessage = decrypted[1+headerLen:]

				log.Println("headerLen", headerLen)

				log.Println("Payload header", payloadHeader, "Payload message", payloadMessage)
				log.Println("Payload header", string(payloadHeader), "Payload message", string(payloadMessage))

				for i := 0; i < len(payloadHeader); i++ {
					log.Printf("Process header byte %4d %1s\n", payloadHeader[i], string(payloadHeader[i]))
				}
			}

		}()

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
