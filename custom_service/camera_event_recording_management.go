package custom_service

import (
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
	"github.com/duncanleo/hc-camera-ffmpeg/custom_characteristic"
)

const TypeCameraEventRecordingManagement = "204"

type CameraEventRecordingManagement struct {
	*service.Service

	Active                                *characteristic.Active
	SupportedCameraRecordingConfiguration *characteristic.SupportedCameraRecordingConfiguration
	SupportedVideoRecordingConfiguration  *characteristic.SupportedVideoRecordingConfiguration
	SupportedAudioRecordingConfiguration  *characteristic.SupportedAudioRecordingConfiguration
	SelectedCameraRecordingConfiguration  *characteristic.SelectedCameraRecordingConfiguration
	RecordingAudioActive                  *custom_characteristic.RecordingAudioActive
}

func NewCameraEventRecordingManagement() *CameraEventRecordingManagement {
	var svc = CameraEventRecordingManagement{}
	svc.Service = service.New(TypeCameraEventRecordingManagement)

	svc.Active = characteristic.NewActive()
	svc.AddCharacteristic(svc.Active.Characteristic)

	svc.SupportedCameraRecordingConfiguration = characteristic.NewSupportedCameraRecordingConfiguration()
	svc.AddCharacteristic(svc.SupportedCameraRecordingConfiguration.Characteristic)

	svc.SupportedVideoRecordingConfiguration = characteristic.NewSupportedVideoRecordingConfiguration()
	svc.AddCharacteristic(svc.SupportedVideoRecordingConfiguration.Characteristic)

	svc.SupportedAudioRecordingConfiguration = characteristic.NewSupportedAudioRecordingConfiguration()
	svc.AddCharacteristic(svc.SupportedAudioRecordingConfiguration.Characteristic)

	svc.SelectedCameraRecordingConfiguration = characteristic.NewSelectedCameraRecordingConfiguration()
	svc.AddCharacteristic(svc.SelectedCameraRecordingConfiguration.Characteristic)

	svc.RecordingAudioActive = custom_characteristic.NewRecordingAudioActive()
	svc.AddCharacteristic(svc.RecordingAudioActive.Characteristic)

	return &svc
}
