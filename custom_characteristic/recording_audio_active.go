package custom_characteristic

import "github.com/brutella/hc/characteristic"

const TypeRecordingAudioActive = "226"

type RecordingAudioActive struct {
	*characteristic.Int
}

func NewRecordingAudioActive() *RecordingAudioActive {
	var char = characteristic.NewInt(TypeRecordingAudioActive)

	char.Format = characteristic.FormatUInt8
	char.Perms = []string{characteristic.PermRead, characteristic.PermWrite, characteristic.PermEvents, PermTimedWrite}
	char.Description = "Recording Audio Active"
	char.MinValue = 0
	char.MaxValue = 1

	char.SetValue(1)

	return &RecordingAudioActive{char}
}
