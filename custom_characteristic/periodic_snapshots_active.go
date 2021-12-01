package custom_characteristic

import "github.com/brutella/hc/characteristic"

const TypePeriodicSnapshotsActive = "225"

type PeriodicSnapshotsActive struct {
	*characteristic.Bool
}

func NewPeriodicSnapshotsActive() *PeriodicSnapshotsActive {
	var char = characteristic.NewBool(TypePeriodicSnapshotsActive)

	char.Format = characteristic.FormatBool
	char.Perms = []string{characteristic.PermRead, characteristic.PermWrite, characteristic.PermEvents}
	char.Description = "Periodic Snapshots Active"

	char.SetValue(true)

	return &PeriodicSnapshotsActive{char}
}
