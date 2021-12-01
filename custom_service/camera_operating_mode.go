package custom_service

import (
	"github.com/brutella/hc/service"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_characteristic"
)

const TypeCameraOperatingMode = "21A"

type CameraOperatingMode struct {
	*service.Service

	EventSnapshotActive     *custom_characteristic.EventSnapshotActive
	HomeKitCameraActive     *custom_characteristic.HomeKitCameraActive
	PeriodicSnapshotsActive *custom_characteristic.PeriodicSnapshotsActive
}

func NewCameraOperatingMode() *CameraOperatingMode {
	var svc = CameraOperatingMode{}
	svc.Service = service.New(TypeCameraOperatingMode)

	svc.EventSnapshotActive = custom_characteristic.NewEventSnapshotActive()
	svc.AddCharacteristic(svc.EventSnapshotActive.Characteristic)

	svc.HomeKitCameraActive = custom_characteristic.NewHomeKitCameraActive()
	svc.AddCharacteristic(svc.HomeKitCameraActive.Characteristic)

	svc.PeriodicSnapshotsActive = custom_characteristic.NewPeriodicSnapshotsActive()
	svc.AddCharacteristic(svc.PeriodicSnapshotsActive.Characteristic)

	return &svc
}
