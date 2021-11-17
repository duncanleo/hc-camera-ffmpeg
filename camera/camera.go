package camera

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/rtp"
	"github.com/brutella/hc/tlv8"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_service"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
	"github.com/duncanleo/hc-camera-ffmpeg/mp4"
)

type EncoderProfile int

const (
	_                  = iota
	CPU EncoderProfile = 1 << (10 * iota)
	VAAPI
)

const (
	protocolWhitelist = "file,udp,tcp,rtp,http,pipe"
)

var (
	debug          = os.Getenv("DEBUG")
	isDebugEnabled = debug == "*" || debug == "ffmpeg"
)

type ServiceConfiguration struct {
	Motion   bool
	Doorbell bool
}

type InputConfiguration struct {
	Source           string
	Format           string
	Audio            bool
	AudioAAC         bool
	TimestampOverlay bool
}

// CreateCamera create a camera accessory
func CreateCamera(accInfo accessory.Info, svcCfg ServiceConfiguration, inputCfg InputConfiguration, encoderProfile EncoderProfile) (*accessory.Camera, func(width, height uint) (*image.Image, error)) {
	go func() {
		for {
			motherStream(inputCfg, encoderProfile)
			log.Println("[MOTHER STREAM] terminated, restarting in 5s")
			time.Sleep(5 * time.Second)
		}
	}()

	camera := accessory.NewCamera(accInfo)

	setupStreamMgmt(inputCfg, camera.StreamManagement1, encoderProfile)
	setupStreamMgmt(inputCfg, camera.StreamManagement2, encoderProfile)

	var cameraOperatingModeService = custom_service.NewCameraOperatingMode()
	camera.AddService(cameraOperatingModeService.Service)

	cameraOperatingModeService.HomeKitCameraActive.OnValueRemoteUpdate(func(value int) {
		log.Printf("CameraOperatingMode HomeKitCameraActive OnValueRemoteUpdate %+v\n", value)
	})

	var dataStreamManagementService = createDataStreamService(inputCfg, encoderProfile)
	camera.AddService(dataStreamManagementService.Service)

	var cameraEventRecordingManagementService = custom_service.NewCameraEventRecordingManagement()

	cameraEventRecordingManagementService.Active.OnValueRemoteUpdate(func(value int) {
		log.Printf("CameraEventRecordingManagement Active OnValueRemoteUpdate %+v\n", value)
	})

	cameraEventRecordingManagementService.SelectedCameraRecordingConfiguration.OnValueRemoteUpdate(func(value []byte) {

		var selection hsv.SelectedCameraRecordingConfiguration
		var err error

		err = tlv8.Unmarshal(value, &selection)

		if err != nil {
			log.Printf("SelectedCameraRecordingConfiguration OnValueRemoteUpdate Error Decoding=%s\n", err)
			return
		}

		SelectedCameraRecordingConfiguration = &selection

		log.Printf("SelectedCameraRecordingConfiguration OnValueRemoteUpdate %+v\n", selection)
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
			PrebufferLength:     int64(hsv.PrebufferLengthStandard / time.Millisecond),
			EventTriggerOptions: hksvEventTriggerBitmask(svcCfg.Motion, svcCfg.Doorbell),
			MediaContainerConfigurations: []hsv.MediaContainerConfiguration{
				{
					MediaContainerType: hsv.MediaContainerTypeFragmentedMP4,
					MediaContainerParameters: []hsv.MediaContainerParameters{
						{
							FragmentLength: int64(hsv.FragmentLengthStandard / time.Millisecond),
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

		inPipe, _ := ffmpegProcess.StdinPipe()

		ffmpegProcess.Stdout = &stdoutPipe
		if isDebugEnabled {
			ffmpegProcess.Stderr = os.Stderr
		}
		err := ffmpegProcess.Start()
		if err != nil {
			return nil, err
		}

		for _, chk := range initChunks {
			dat, _ := chk.Assemble()

			inPipe.Write(dat)
		}

		var prebufferDataClone = make([]mp4.Chunk, len(prebufferData))
		copy(prebufferDataClone, prebufferData)

		for _, chk := range prebufferDataClone {
			dat, _ := chk.Assemble()

			inPipe.Write(dat)
		}

		err = ffmpegProcess.Wait()
		if err != nil {
			return nil, err
		}

		img, _, err := image.Decode(&stdoutPipe)
		return &img, err
	}

	return camera, snapshot
}
