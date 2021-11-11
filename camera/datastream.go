package camera

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os/exec"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/hap"
	"github.com/brutella/hc/tlv8"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_service"
	"github.com/duncanleo/hc-camera-ffmpeg/hds"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
	"github.com/duncanleo/hc-camera-ffmpeg/mp4"
)

var (
	HAPContext                           *hap.Context
	SelectedCameraRecordingConfiguration *hsv.SelectedCameraRecordingConfiguration

	sessionMap = make(map[string]*exec.Cmd)
)

func createDataStreamService(inputCfg InputConfiguration, encoderProfile EncoderProfile) *custom_service.DataStreamTransportManagement {
	var svc = custom_service.NewDataStreamTransportManagement()

	setTLV8Payload(
		svc.SupportedDataStreamTransportConfiguration.Bytes,
		hsv.SupportedDataStreamTransportConfiguration{
			TransferTransportConfigurations: []hsv.TransferTransportConfiguration{
				{
					TransportType: hsv.TransportTypeHomeKitDataStream,
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
				var event = hdsPayload.Header["event"]

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

							if SelectedCameraRecordingConfiguration == nil {
								log.Println("[HKSV] SelectedCameraRecordingConfiguration not written, cannot begin encoding")
								return
							}

							// TODO: Start sending MP4 fragments
							var args = generateHKSVArguments(inputCfg, encoderProfile, *SelectedCameraRecordingConfiguration)
							var ffmpegProcess = exec.Command(
								"ffmpeg",
								args...,
							)
							log.Println(ffmpegProcess.String())
							//		ffmpegProcess.Stderr = os.Stdout
							ffmpegOut, err := ffmpegProcess.StdoutPipe()
							if err != nil {
								log.Println(err)
								return
							}

							ffmpegOutBuffer := bufio.NewReaderSize(ffmpegOut, 1000000)

							go func() {
								defer ffmpegOut.Close()

								if ffmpegProcess.ProcessState != nil && ffmpegProcess.ProcessState.Exited() {
									return
								}

								var dataSequenceNumber = 1
								var dataChunkSequenceNumber = 1
								var fragmentTotal = int(hsv.FragmentLengthStandard/FragmentDuration) / 2
								var collectedChunks = make([]mp4.Chunk, 0)

								for {
									log.Println("[FFMPEG HKSV] Waiting for data. Buffer size=", ffmpegOutBuffer.Buffered())

									var chunkHeader = make([]byte, 8)
									_, err = io.ReadFull(ffmpegOutBuffer, chunkHeader)
									if err != nil {
										log.Println(err)
										return
									}

									log.Println("[FFMPEG HKSV] Read chunk header", chunkHeader)

									var chunkTypeBytes = chunkHeader[4:]
									var chunkType = string(chunkTypeBytes)

									var prependChunkData = make([]byte, 0)

									if chunkType == "ftyp" {
										// Must read the sub-type as well
										prependChunkData = make([]byte, 4)

										_, err = io.ReadFull(ffmpegOutBuffer, prependChunkData)
										if err != nil {
											log.Println(err)
											return
										}

										log.Println("[FFMPEG HKSV] Read chunk sub-type", string(prependChunkData))
									}

									var chunkSizeBytes = chunkHeader[0:4]
									var chunkCompleteSize = binary.BigEndian.Uint32(chunkSizeBytes)
									var chunkSize = chunkCompleteSize - uint32(len(chunkHeader)) - uint32(len(prependChunkData))

									log.Printf("[FFMPEG HKSV] Chunk type=%s size=%d", chunkType, chunkSize)

									// Sanity check
									switch chunkType {
									case "ftyp", "mdat", "moov", "pnot", "udta", "uuid", "moof", "free", "skip", "jP2 ", "wide", "load", "ctab", "imap", "matt", "kmat", "clip", "crgn", "sync", "chap", "tmcd", "scpt", "ssrc", "PICT":
									default:
										log.Println("[FFMPEG HKSV] Unknown chunk type", chunkType, "discarding")
										discardCount, err := ffmpegOutBuffer.Discard(int(chunkSize))
										if err != nil {
											log.Println(err)
											return
										}
										log.Println("[FFMPEG HKSV] Discarded", discardCount, "bytes")
										continue
									}

									var chunkData = make([]byte, chunkSize)
									_, err = io.ReadFull(ffmpegOutBuffer, chunkData)
									if err != nil {
										log.Println(err)
										return
									}

									var chunk = mp4.Chunk{
										Size:     chunkCompleteSize,
										MainType: chunkType,
										SubType:  string(prependChunkData),
										Data:     chunkData,
									}

									collectedChunks = append(collectedChunks, chunk)

									var isInitChunk = collectedChunks[0].MainType == "ftyp"

									if len(collectedChunks) == 2 && (isInitChunk || dataChunkSequenceNumber <= fragmentTotal) {

										// Time to flush
										var dataType = "mediaFragment"

										if isInitChunk {
											dataType = "mediaInitialization"
										}

										var isLastDataChunk = isInitChunk || dataChunkSequenceNumber == fragmentTotal

										var collectedChunkTypes = make([]string, 0)
										var data = make([]byte, 0)
										for _, chk := range collectedChunks {
											collectedChunkTypes = append(collectedChunkTypes, chk.MainType)

											dat, err := chk.Assemble()

											if err != nil {
												log.Println(err)
												return
											}

											data = append(data, dat...)
										}

										var payload = hds.Payload{
											Header: map[string]interface{}{
												"protocol": "dataSend",
												"event":    "data",
											},
											Message: map[string]interface{}{
												"status":   hds.StatusSuccess,
												"streamId": message["streamId"],
												"packets": []interface{}{
													map[string]interface{}{
														"data": data,
														"metadata": map[string]interface{}{
															"dataType":                dataType,
															"dataSequenceNumber":      dataSequenceNumber,
															"dataChunkSequenceNumber": dataChunkSequenceNumber,
															"isLastDataChunk":         isLastDataChunk,
														},
													},
												},
											},
										}

										log.Printf("[FFMPEG HKSV] Flushing chunks %+v dataType=%s dataSequenceNumber=%d dataChunkSequenceNumber=%d fragmentTotal=%d isLastDataChunk=%+v\n", collectedChunkTypes, dataType, dataSequenceNumber, dataChunkSequenceNumber, fragmentTotal, isLastDataChunk)

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

										dataChunkSequenceNumber++
										collectedChunks = make([]mp4.Chunk, 0)

										if isLastDataChunk {
											dataSequenceNumber++
											dataChunkSequenceNumber = 1
										}
									}

								}
							}()

							err = ffmpegProcess.Start()
							if err != nil {
								log.Println(err)
								return
							}

							log.Println("[FFMPEG HKSV] Spawn PID", ffmpegProcess.Process.Pid)

							defer func() {
								if ffmpegProcess.ProcessState != nil && !ffmpegProcess.ProcessState.Exited() {
									log.Println("[FFMPEG HKSV] Terminating PID", ffmpegProcess.Process.Pid)
									ffmpegProcess.Process.Kill()
								}
							}()

							sessionMap[tcpConn.RemoteAddr().String()] = ffmpegProcess
						}
					}

					switch event {
					case "close":
						// TODO: Handle close
						if ffmpegProcess, ok := sessionMap[tcpConn.RemoteAddr().String()]; ok {
							log.Println("[FFMPEG HKSV] Terminating PID", ffmpegProcess.Process.Pid)
							ffmpegProcess.Process.Kill()
						}
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
