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

		var pairVerifyHandler = session.PairVerifyHandler()
		var sharedKey = pairVerifyHandler.SharedKey()

		sess, err := hds.NewHDSSession(request.ControllerKeySalt, accessoryKeySalt, sharedKey)
		if err != nil {
			log.Println(err)
			return
		}

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
				log.Println("[TCP] Ended TCP listener at", listenerAddress)
			}()

			log.Println("[TCP] Started TCP listener at", listenerAddress)

			tcpConn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				return
			}

			defer tcpConn.Close()

			for {
				var err error

				frame, err := hds.NewFrame(tcpConn)
				if err != nil {
					log.Println(err)
					return
				}

				log.Printf("[TCP] Recv frame %+v\n", frame)

				decrypted, err := sess.Decrypt(frame.Payload, frame.AuthTag, frame.Header[:])
				if err != nil {
					log.Println(err)
					return
				}

				hdsPayload, err := hds.NewPayload(decrypted)
				if err != nil {
					log.Println(err)
					return
				}

				decodedHeader, err := hdsPayload.DecodedHeader()
				if err != nil {
					log.Println(err)
					return
				}

				decodedMessage, err := hdsPayload.DecodedMessage()
				if err != nil {
					log.Println(err)
					return
				}

				var decodedHeaderMap = decodedHeader.(map[string]interface{})
				var protocol = decodedHeaderMap["protocol"]

				log.Printf("[HDS] New message with protocol=%s: %+v\n", protocol, decodedMessage)

				switch protocol {
				case "control":

					switch decodedHeaderMap["request"] {
					case "hello":
						log.Println("[HDS] Received HELLO!")

						// Respond with the same hello message
						var respPayloadHeader = map[string]interface{}{
							"protocol": "control",
							"response": "hello",
							"id":       decodedHeaderMap["id"],
							"status":   hds.StatusSuccess,
						}

						encodedHeader, err := hds.EncodeDataFormat(respPayloadHeader)
						if err != nil {
							log.Println(err)
							return
						}

						var respPayloadMessage = map[string]interface{}{}

						encodedMessage, err := hds.EncodeDataFormat(respPayloadMessage)
						if err != nil {
							log.Println(err)
							return
						}

						var respPayload []byte
						var encodedHeaderLen = len(encodedHeader)

						respPayload = append(respPayload, byte(encodedHeaderLen))
						respPayload = append(respPayload, encodedHeader...)
						respPayload = append(respPayload, encodedMessage...)

						var respPayloadLen = len(respPayload)

						var newHeader = make([]byte, 4)
						binary.BigEndian.PutUint32(newHeader, uint32(respPayloadLen))
						newHeader[0] = 0x01

						encrypted, mac, err := sess.Encrypt(respPayload, newHeader)
						if err != nil {
							log.Println(err)
							return
						}

						var packet []byte
						packet = append(packet, newHeader...)
						packet = append(packet, encrypted...)
						packet = append(packet, mac[:]...)

						log.Println("[TCP] Writing HELLO response")

						tcpConn.Write(packet)
					}
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
