package hsv

import "time"

const (
	VideoCodecH264 = 0
	VideoCodecH265 = 1

	AudioRecordingCodecAAC_LC  = 0
	AudioRecordingCodecAAC_ELD = 1

	AudioRecordingSampleRate8Khz  = 0
	AudioRecordingSampleRate16Khz = 1
	AudioRecordingSampleRate24Khz = 2
	AudioRecordingSampleRate32Khz = 3
	AudioRecordingSampleRate44Khz = 4
	AudioRecordingSampleRate48Khz = 5

	SetupDataStreamTransportCommandStart = 0

	SetupDataStreamTransportStatusSuccess      = 0
	SetupDataStreamTransportStatusGenericError = 0
	SetupDataStreamTransportStatusBusy         = 0

	TransportTypeHomeKitDataStream = 0

	MediaContainerTypeFragmentedMP4 = 0

	PrebufferLengthStandard = 4000 * time.Millisecond
	FragmentLengthStandard  = 4000 * time.Millisecond
)
