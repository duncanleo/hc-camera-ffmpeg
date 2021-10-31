package camera

import (
	"crypto/rand"
	"encoding/base64"
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

				frame, err := hds.NewFrameFromReader(tcpConn)
				if err != nil {
					log.Println(err)
					return
				}

				log.Printf("[TCP] Recv frame with %d-byte payload\n", len(frame.Payload))

				decrypted, err := sess.Decrypt(frame.Payload, frame.AuthTag, frame.Header[:])
				if err != nil {
					log.Println(err)
					return
				}

				hdsPayload, err := hds.NewPayloadFromRaw(decrypted)
				if err != nil {
					log.Println(err)
					return
				}

				var protocol = hdsPayload.Header["protocol"]
				var request = hdsPayload.Header["request"]

				log.Printf("[HDS] New message with protocol=%s header=%+v message=%+v\n", protocol, hdsPayload.Header, hdsPayload.Message)

				switch protocol {
				case "control":
					log.Printf("[HDS] Received control with request=%s\n", request)

					switch request {
					case "hello":

						var payload = hds.Payload{
							Header: map[string]interface{}{
								"protocol": "control",
								"response": "hello",
								"id":       hdsPayload.Header["id"],
								"status":   hds.StatusSuccess,
							},
							Message: map[string]interface{}{},
						}

						encodedPayload, err := payload.Encode()
						if err != nil {
							log.Println(err)
							return
						}

						newFrame, err := hds.NewFrameFromPayloadAndSession(encodedPayload, &sess)
						if err != nil {
							log.Println(err)
							return
						}

						log.Println("[TCP] Writing HELLO response")

						tcpConn.Write(newFrame.Assemble())
					}
				case "dataSend":
					var message = hdsPayload.Message.(map[string]interface{})
					var dataSendType = message["type"]
					log.Printf("[HDS] Received dataSend with type=%s\n", dataSendType)

					switch request {
					case "open":
						switch dataSendType {
						case "ipcamera.recording":
							var payload = hds.Payload{
								Header: map[string]interface{}{
									"protocol": "dataSend",
									"id":       hdsPayload.Header["id"],
									"response": "open",
									"status":   hds.StatusSuccess,
								},
								Message: map[string]interface{}{
									"status": hds.StatusSuccess,
								},
							}

							encodedPayload, err := payload.Encode()
							if err != nil {
								log.Println(err)
								return
							}

							newFrame, err := hds.NewFrameFromPayloadAndSession(encodedPayload, &sess)
							if err != nil {
								log.Println(err)
								return
							}

							tcpConn.Write(newFrame.Assemble())

							log.Println("[TCP] Writing dataSend response")

							// TODO: Start sending MP4 fragments
						}
					case "close":
						// TODO: Handle close
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

		log.Printf("[HC] Setting Write Response %+v\n", response)

		encoded, err := tlv8.Marshal(response)
		if err != nil {
			log.Println(err)
			return
		}

		c.WriteResponse = encoded

	})

	return svc
}
