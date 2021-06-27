package custom_characteristic

import "github.com/brutella/hc/characteristic"

const TypeHomeKitCameraActive = "21B"

type HomeKitCameraActive struct {
	*characteristic.Int
}

func NewHomeKitCameraActive() *HomeKitCameraActive {
	var char = characteristic.NewInt(TypeHomeKitCameraActive)

	char.Format = characteristic.FormatUInt8
	char.Perms = []string{characteristic.PermRead, characteristic.PermWrite, characteristic.PermEvents}
	char.Description = "HomeKit Camera Active"
	char.MinValue = 0
	char.MaxValue = 1

	char.SetValue(1)

	return &HomeKitCameraActive{char}
}
