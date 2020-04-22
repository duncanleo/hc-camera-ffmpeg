package camera

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/rtp"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/tlv8"
	"github.com/r3labs/diff"
)

type EncoderProfile int

const (
	_                  = iota
	CPU EncoderProfile = 1 << (10 * iota)
	OMX
	VAAPI
)

const (
	protocolWhitelist = "file,udp,tcp,rtp,http"
)

var (
	debug          = os.Getenv("DEBUG")
	isDebugEnabled = debug == "*" || debug == "ffmpeg"
)

type InputConfiguration struct {
	Source           string
	Format           string
	Audio            bool
	AudioAAC         bool
	TimestampOverlay bool
}

func generateArguments(inputCfg InputConfiguration, streamCfg rtp.StreamConfiguration, se rtp.SetupEndpoints, encoderProfile EncoderProfile) []string {
	var inputOpts []string
	var encoder string
	var encoderOpts []string
	switch encoderProfile {
	case CPU:
		encoder = "h264"
		encoderOpts = []string{
			"-x264-params",
			"intra-refresh=1:bframes=0",
			"-vf",
			fmt.Sprintf("scale=%d:-1", streamCfg.Video.Attributes.Width),
		}
		break
	case OMX:
		encoder = "h264_omx"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("scale=%d:-1", streamCfg.Video.Attributes.Width),
		}
		break
	case VAAPI:
		inputOpts = []string{
			"-vaapi_device",
			"/dev/dri/renderD128",
			"-hwaccel",
			"vaapi",
		}
		encoder = "h264_vaapi"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("format=nv12|vaapi,hwupload,scale_vaapi=w=%d:h=-1", streamCfg.Video.Attributes.Width),
			"-quality",
			"10",
			"-bf",
			"0",
		}
		break
	}
	var args []string
	args = append(args, inputOpts...)
	args = append(
		args,
		"-f",
		inputCfg.Format,
		"-protocol_whitelist",
		protocolWhitelist,
		"-i",
		inputCfg.Source,
		"-c:v",
		encoder,
		"-profile:v",
		streamVideoProfile(streamCfg),
		"-level:v",
		streamVideoCodecLevel(streamCfg),
		"-vsync",
		"vfr",
	)

	args = append(args, encoderOpts...)
	args = append(args,
		"-preset",
		"veryfast",
	)

	if inputCfg.TimestampOverlay {
		args = append(
			args,
			"-filter_complex",
			"drawtext=text='time\\: %{localtime}':fontcolor=white",
		)
	}

	args = append(
		args,
		"-payload_type",
		fmt.Sprintf("%d", streamCfg.Video.RTP.PayloadType),
		"-ssrc",
		"1",
		"-map",
		"0:0",
		"-f",
		"rtp",
		"-srtp_out_suite",
		"AES_CM_128_HMAC_SHA1_80",
		"-b:v",
		fmt.Sprintf("%dk", streamCfg.Video.RTP.Bitrate),
		"-srtp_out_params",
		se.Video.SrtpKey(),
		fmt.Sprintf("srtp://%s:%d?rtcpport=%d&localrtcpport=%d&pkt_size=%d&timeout=60", se.ControllerAddr.IPAddr, se.ControllerAddr.VideoRtpPort, se.ControllerAddr.VideoRtpPort, se.ControllerAddr.VideoRtpPort, streamVideoMTP(se)),
	)

	var audioCodec = "libopus"
	if inputCfg.AudioAAC {
		audioCodec = streamAudioCodec(streamCfg)
	}

	if inputCfg.Audio {
		args = append(
			args,
			"-payload_type",
			fmt.Sprintf("%d", streamCfg.Audio.RTP.PayloadType),
			"-ssrc",
			"2",
			"-c:a",
			audioCodec,
			"-map",
			"0:1",
			"-f",
			"rtp",
			"-ar",
			fmt.Sprintf("%d", streamAudioSampleRate(streamCfg)),
		)
		args = append(args, streamAudioCodecOptions(streamCfg)...)
		args = append(
			args,
			"-srtp_out_suite",
			"AES_CM_128_HMAC_SHA1_80",
			"-b:a",
			fmt.Sprintf("%dk", streamCfg.Audio.RTP.Bitrate),
			"-frame_duration",
			"20",
			"-srtp_out_params",
			se.Audio.SrtpKey(),
			fmt.Sprintf("srtp://%s:%d?rtcpport=%d&localrtcpport=%d&pkt_size=%d&timeout=60", se.ControllerAddr.IPAddr, se.ControllerAddr.AudioRtpPort, se.ControllerAddr.AudioRtpPort, se.ControllerAddr.AudioRtpPort, 3768),
		)
	}

	return args
}

func generateSnapshotArguments(inputCfg InputConfiguration, width uint) []string {
	var args = []string{
		"-f",
		inputCfg.Format,
		"-protocol_whitelist",
		protocolWhitelist,
		"-i",
		inputCfg.Source,
		"-c:v",
		"png",
		"-vframes",
		"1",
		"-vsync",
		"vfr",
		"-video_size",
		fmt.Sprintf("%d:-2", width),
		"-f",
		"image2pipe",
		"-",
	}
	return args
}

func setupStreamMgmt(inputCfg InputConfiguration, sm *service.CameraRTPStreamManagement, encoderProfile EncoderProfile) {
	setTLV8Payload(sm.StreamingStatus.Bytes, rtp.StreamingStatus{Status: rtp.StreamingStatusAvailable})
	setTLV8Payload(sm.SupportedVideoStreamConfiguration.Bytes, rtp.DefaultVideoStreamConfiguration())

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

// CreateCamera create a camera accessory
func CreateCamera(accInfo accessory.Info, inputCfg InputConfiguration, encoderProfile EncoderProfile) (*accessory.Camera, func(width, height uint) (*image.Image, error)) {
	camera := accessory.NewCamera(accInfo)

	setupStreamMgmt(inputCfg, camera.StreamManagement1, encoderProfile)
	setupStreamMgmt(inputCfg, camera.StreamManagement2, encoderProfile)

	var snapshot = func(width, height uint) (*image.Image, error) {
		var args = generateSnapshotArguments(inputCfg, width)

		var ffmpegProcess = exec.Command(
			"ffmpeg",
			args...,
		)
		var stdoutPipe bytes.Buffer

		log.Println(ffmpegProcess.String())

		ffmpegProcess.Stdout = &stdoutPipe
		if isDebugEnabled {
			ffmpegProcess.Stderr = os.Stderr
		}
		err := ffmpegProcess.Run()
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(&stdoutPipe)
		return &img, err
	}

	return camera, snapshot
}

func setTLV8Payload(c *characteristic.Bytes, v interface{}) {
	if tlv8, err := tlv8.Marshal(v); err == nil {
		c.SetValue(tlv8)
	} else {
		log.Fatal(err)
	}
}
