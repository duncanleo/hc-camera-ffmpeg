package camera

import (
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/rtp"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/tlv8"
	"github.com/r3labs/diff"
)

func setupStreamMgmt(inputCfg InputConfiguration, sm *service.CameraRTPStreamManagement, encoderProfile EncoderProfile) {
	setTLV8Payload(sm.StreamingStatus.Bytes, rtp.StreamingStatus{Status: rtp.StreamingStatusAvailable})
	setTLV8Payload(sm.SupportedVideoStreamConfiguration.Bytes, rtp.DefaultVideoStreamConfiguration())

	var active = characteristic.NewActive()
	active.SetValue(1)
	sm.AddCharacteristic(active.Characteristic)
	active.OnValueRemoteUpdate(func(value int) {
		log.Printf("CameraRTPStreamManagement Active OnValueRemoteUpdate %+v\n", value)
	})

	var supportedCodecs = []rtp.AudioCodecConfiguration{
		rtp.NewOpusAudioCodecConfiguration(),
	}

	if inputCfg.AudioAAC {
		supportedCodecs = append(supportedCodecs, rtp.NewAacEldAudioCodecConfiguration())
	}

	setTLV8Payload(sm.SupportedAudioStreamConfiguration.Bytes, rtp.AudioStreamConfiguration{
		Codecs:       supportedCodecs,
		ComfortNoise: false,
	})
	setTLV8Payload(sm.SupportedRTPConfiguration.Bytes, rtp.NewConfiguration(rtp.CryptoSuite_AES_CM_128_HMAC_SHA1_80))

	var ffmpegProcess *exec.Cmd
	var sessionMap = make(map[string]rtp.SetupEndpoints)
	var initialStartMap = make(map[string]rtp.StreamConfiguration)

	sm.SelectedRTPStreamConfiguration.OnValueRemoteUpdate(func(value []byte) {
		var cfg rtp.StreamConfiguration
		var err error
		err = tlv8.Unmarshal(value, &cfg)
		if err != nil {
			log.Fatalf("SelectedRTPStreamConfiguration: Could not unmarshal tlv8 data: %s\n", err)
		}

		se := sessionMap[string(cfg.Command.Identifier)]

		switch cfg.Command.Type {
		case rtp.SessionControlCommandTypeStart:
			if isDebugEnabled {
				log.Println("start")
			}

			args := generateArguments(inputCfg, cfg, se, encoderProfile)

			ffmpegProcess = exec.Command(
				"ffmpeg",
				args...,
			)

			if isDebugEnabled {
				log.Println(ffmpegProcess.String())
			}

			if isDebugEnabled {
				ffmpegProcess.Stdout = os.Stdout
				ffmpegProcess.Stderr = os.Stderr
			}
			err = ffmpegProcess.Start()
			initialStartMap[string(cfg.Command.Identifier)] = cfg

			log.Printf("start: %+v\n", cfg)
			break
		case rtp.SessionControlCommandTypeResume:
			err = ffmpegProcess.Process.Signal(syscall.SIGCONT)
			log.Println("resume")
			break
		case rtp.SessionControlCommandTypeReconfigure:
			if isDebugEnabled {
				changelog, _ := diff.Diff(initialStartMap[string(cfg.Command.Identifier)], cfg)
				log.Printf("reconfigure: %+v\n", changelog)
			}
			break
		case rtp.SessionControlCommandTypeSuspend:
			log.Println("suspend")
			err = ffmpegProcess.Process.Signal(syscall.SIGSTOP)
			break
		case rtp.SessionControlCommandTypeEnd:
			log.Println("end")
			err = ffmpegProcess.Process.Kill()

			delete(sessionMap, string(cfg.Command.Identifier))
			delete(initialStartMap, string(cfg.Command.Identifier))
			break
		}

		if err != nil && isDebugEnabled {
			log.Println(err)
		}

	})

	sm.SetupEndpoints.OnValueUpdateFromConn(func(conn net.Conn, c *characteristic.Characteristic, new, old interface{}) {
		buf := sm.SetupEndpoints.GetValue()
		var req rtp.SetupEndpoints
		err := tlv8.Unmarshal(buf, &req)
		if err != nil && isDebugEnabled {
			log.Fatalf("SetupEndpoints: Could not unmarshal tlv8 data: %s\n", err)
		}

		log.Printf("SetupEndpoints: %+v\n", req)

		ssrcVideo := int32(1)
		ssrcAudio := int32(2)

		var ip string

		switch addr := conn.LocalAddr().(type) {
		case *net.TCPAddr:
			ip = addr.IP.String()
			break
		case *net.UDPAddr:
			ip = addr.IP.String()
			break
		}

		if isDebugEnabled {
			log.Printf("[SetupEndpoints] IPv%d %s VideoRtpPort=%d AudioRtpPort=%d\n", req.ControllerAddr.IPVersion, ip, req.ControllerAddr.VideoRtpPort, req.ControllerAddr.AudioRtpPort)
		}

		resp := rtp.SetupEndpointsResponse{
			SessionId: req.SessionId,
			Status:    rtp.SessionStatusSuccess,
			AccessoryAddr: rtp.Addr{
				IPVersion:    req.ControllerAddr.IPVersion,
				IPAddr:       ip,
				VideoRtpPort: req.ControllerAddr.VideoRtpPort,
				AudioRtpPort: req.ControllerAddr.AudioRtpPort,
			},
			Video:     req.Video,
			Audio:     req.Audio,
			SsrcVideo: ssrcVideo,
			SsrcAudio: ssrcAudio,
		}

		sessionMap[string(req.SessionId)] = req

		setTLV8Payload(sm.SetupEndpoints.Bytes, resp)
	})
}
