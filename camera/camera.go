package camera

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/rtp"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_service"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
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

// CreateCamera create a camera accessory
func CreateCamera(accInfo accessory.Info, inputCfg InputConfiguration, encoderProfile EncoderProfile) (*accessory.Camera, func(width, height uint) (*image.Image, error)) {
	camera := accessory.NewCamera(accInfo)

	setupStreamMgmt(inputCfg, camera.StreamManagement1, encoderProfile)
	setupStreamMgmt(inputCfg, camera.StreamManagement2, encoderProfile)

	var cameraOperatingModeService = custom_service.NewCameraOperatingMode()
	camera.AddService(cameraOperatingModeService.Service)

	cameraOperatingModeService.HomeKitCameraActive.OnValueRemoteUpdate(func(value int) {
		log.Printf("CameraOperatingMode HomeKitCameraActive OnValueRemoteUpdate %+v\n", value)
	})

	var dataStreamManagementService = custom_service.NewDataStreamTransportManagement()

	setTLV8Payload(
		dataStreamManagementService.SupportedDataStreamTransportConfiguration.Bytes,
		hsv.SupportedDataStreamTransportConfiguration{
			TransferTransportConfigurations: []hsv.TransferTransportConfiguration{
				{
					TransportType: 0,
				},
			},
		})

	dataStreamManagementService.SetupDataStreamTransport.OnValueRemoteUpdate(func(v []byte) {
		log.Printf("SetupDataStreamTransport onValueRemoteUpdate %+v\n", v)
	})

	dataStreamManagementService.SetupDataStreamTransport.OnValueUpdate(func(c *characteristic.Characteristic, newValue, oldValue interface{}) {
		log.Printf("SetupDataStreamTransport OnValueUpdate %+v %+v\n", newValue, oldValue)
	})

	dataStreamManagementService.SetupDataStreamTransport.OnValueUpdateFromConn(func(conn net.Conn, c *characteristic.Characteristic, newValue, oldValue interface{}) {
		log.Printf("SetupDataStreamTransport OnValueUpdateFromConn %+v %+v\n", newValue, oldValue)
	})

	camera.AddService(dataStreamManagementService.Service)

	var cameraEventRecordingManagementService = custom_service.NewCameraEventRecordingManagement()

	cameraEventRecordingManagementService.Active.OnValueRemoteUpdate(func(value int) {
		log.Printf("CameraEventRecordingManagement Active OnValueRemoteUpdate %+v\n", value)
	})

	cameraEventRecordingManagementService.SelectedCameraRecordingConfiguration.OnValueRemoteUpdate(func(value []byte) {
		log.Printf("SelectedCameraRecordingConfiguration OnValueRemoteUpdate %+v\n", value)
	})

	cameraEventRecordingManagementService.AddLinkedService(dataStreamManagementService.Service)
	camera.AddService(cameraEventRecordingManagementService.Service)

	setTLV8Payload(
		cameraEventRecordingManagementService.SupportedVideoRecordingConfiguration.Bytes,
		hsv.SupportedVideoRecordingConfiguration{
			CodecConfiguration: []hsv.VideoConfiguration{
				{
					Codec: hsv.VideoCodecH264,
					VideoCodecParameters: []hsv.VideoCodecParameters{
						{
							ProfileID: rtp.VideoCodecProfileConstrainedBaseline,
							Level:     rtp.VideoCodecLevel3_1,
						},
						{
							ProfileID: rtp.VideoCodecProfileMain,
							Level:     rtp.VideoCodecLevel3_1,
						},
						{
							ProfileID: rtp.VideoCodecProfileHigh,
							Level:     rtp.VideoCodecLevel3_1,
						},
					},
					VideoAttributes: []hsv.VideoAttributes{
						// {
						// 	ImageWidth:  1280,
						// 	ImageHeight: 720,
						// 	FrameRate:   15,
						// },
						{
							ImageWidth:  1280,
							ImageHeight: 720,
							FrameRate:   30,
						},
						// {
						// 	ImageWidth:  1920,
						// 	ImageHeight: 1080,
						// 	FrameRate:   15,
						// },
						{
							ImageWidth:  1920,
							ImageHeight: 1080,
							FrameRate:   30,
						},
					},
				},
			},
		})

	setTLV8Payload(
		cameraEventRecordingManagementService.SupportedAudioRecordingConfiguration.Bytes,
		hsv.SupportedAudioRecordingConfiguration{
			CodecConfiguration: []hsv.AudioConfiguration{
				{
					Codec: hsv.AudioRecordingCodecAAC_LC, // AAC-LC
					AudioCodecParameters: []hsv.AudioCodecParameters{
						{
							Channels:     1,
							BitrateModes: rtp.AudioCodecBitrateConstant,
							SampleRates: []byte{
								// NOTE: Somehow 32Khz is what gets incoming Data Stream requests working
								hsv.AudioRecordingSampleRate32Khz,
							},
						},
					},
				},
			},
		})

	setTLV8Payload(
		cameraEventRecordingManagementService.SupportedCameraRecordingConfiguration.Bytes,
		hsv.RecordingConfiguration{
			PrebufferLength:     4000,
			EventTriggerOptions: 0x01,
			MediaContainerConfigurations: []hsv.MediaContainerConfiguration{
				{
					MediaContainerType: 0, // Fragmented MP4
					MediaContainerParameters: []hsv.MediaContainerParameters{
						{
							FragmentLength: 4000,
						},
					},
				},
			},
		})

	var snapshot = func(width, height uint) (*image.Image, error) {
		var args = generateSnapshotArguments(inputCfg, width)

		var ffmpegProcess = exec.Command(
			"ffmpeg",
			args...,
		)
		var stdoutPipe bytes.Buffer

		if isDebugEnabled {
			log.Println(ffmpegProcess.String())
		}

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
