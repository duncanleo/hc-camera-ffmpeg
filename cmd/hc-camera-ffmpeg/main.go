package main

import (
	"flag"
	"log"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/duncanleo/hc-camera-ffmpeg/camera"
)

func main() {
	var port = flag.String("port", "", "port for the HC accessory, leave empty to randomise")
	var pin = flag.String("pin", "00102003", "pairing PIN for the accessory")
	var storagePath = flag.String("storagePath", "hc-camera-ffmpeg-storage", "storage path")

	var name = flag.String("name", "HomeKit Camera", "name for the HomeKit Camera")
	var manufacturer = flag.String("manufacturer", "Raspberry Pi Foundation", "manufacturer for the HomeKit Camera")
	var model = flag.String("model", "Camera Module", "model for the HomeKit Camera")

	var cameraInput = flag.String("cameraInput", "/dev/video0", "input for the camera")
	var cameraFormat = flag.String("cameraFormat", "v4l2", "input for the camera")
	var cameraAudio = flag.Bool("cameraAudio", false, "whether the camera has audio")
	var encoderProfile = flag.String("encoderProfile", "CPU", "encoder profile for FFMPEG. Accepts: CPU, OMX, VAAPI")
	var audioAAC = flag.Bool("aac", false, "whether to enable the libfdk-aac codec")
	var timestampOverlay = flag.Bool("timestampOverlay", false, "whether to enable timestamp overlay in FFMPEG")

	flag.Parse()

	hcConfig := hc.Config{
		Pin:         *pin,
		StoragePath: *storagePath,
		Port:        *port,
	}

	cameraInfo := accessory.Info{
		Name:         *name,
		Manufacturer: *manufacturer,
		Model:        *model,
	}

	var encProfile = camera.CPU
	switch *encoderProfile {
	case "OMX":
		encProfile = camera.OMX
		break
	case "VAAPI":
		encProfile = camera.VAAPI
		break
	}

	var inputCfg = camera.InputConfiguration{
		Source:           *cameraInput,
		Format:           *cameraFormat,
		Audio:            *cameraAudio,
		AudioAAC:         *audioAAC,
		TimestampOverlay: *timestampOverlay,
	}

	cameraAcc, snapshotFunc := camera.CreateCamera(cameraInfo, inputCfg, encProfile)

	t, err := hc.NewIPTransport(hcConfig, cameraAcc.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	t.CameraSnapshotReq = snapshotFunc

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
