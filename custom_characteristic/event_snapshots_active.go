package custom_characteristic

import "github.com/brutella/hc/characteristic"

const TypeEventSnapshotActive = "223"

type EventSnapshotActive struct {
	*characteristic.Bool
}

func NewEventSnapshotActive() *EventSnapshotActive {
	var char = characteristic.NewBool(TypeEventSnapshotActive)

	char.Format = characteristic.FormatBool
	char.Perms = []string{characteristic.PermRead, characteristic.PermWrite, characteristic.PermEvents}
	char.Description = "Event Snapshots Active"

	char.SetValue(false)

	return &EventSnapshotActive{char}
}
