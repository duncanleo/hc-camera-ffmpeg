package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
	"github.com/duncanleo/hc-camera-ffmpeg/camera"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func connect(clientID string, uri *url.URL) (mqtt.Client, error) {
	var opts = mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)
	opts.CleanSession = false

	var client = mqtt.NewClient(opts)
	var token = client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	return client, token.Error()
}

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

	var brokerURI = flag.String("brokerURI", "mqtt://127.0.0.1:1883", "URI of the MQTT broker")
	var clientID = flag.String("clientID", "hc-camera-ffmpeg", "client ID for MQTT")

	var doorbell = flag.Bool("doorbell", false, "whether to enable video doorbell support")
	var doorbellTopic = flag.String("doorbellTopic", "rpi-mqtt-doorbell", "MQTT topic to subscribe to")

	var motion = flag.Bool("motion", false, "whether to enable motion support")
	var motionTopic = flag.String("motionTopic", "mqtt-publish", "MQTT topic to subscribe to")

	flag.Parse()

	if *doorbell {
	}

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

	var mqttURI *url.URL
	var client mqtt.Client
	var err error

	if *doorbell || *motion {
		mqttURI, err = url.Parse(*brokerURI)
		if err != nil {
			log.Fatal(err)
		}

		client, err = connect(*clientID, mqttURI)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *doorbell {
		var doorbellService = service.NewDoorbell()
		cameraAcc.AddService(doorbellService.Service)

		client.Subscribe(*doorbellTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
			log.Printf("[%s]: %s\n", *doorbellTopic, string(msg.Payload()))
			doorbellService.ProgrammableSwitchEvent.SetValue(characteristic.ProgrammableSwitchEventLongPress)
			doorbellService.ProgrammableSwitchEvent.UpdateValue(characteristic.ProgrammableSwitchEventSinglePress)
		})
	}

	if *motion {
		var motionService = service.NewMotionSensor()
		cameraAcc.AddService(motionService.Service)

		client.Subscribe(*motionTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
			log.Printf("[%s]: %s\n", *motionTopic, string(msg.Payload()))
			motionService.MotionDetected.Bool.SetValue(string(msg.Payload()) == "ON")
		})
	}

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
