package hsv

type SelectedCameraRecordingConfiguration struct {
	SelectedGeneralConfiguration RecordingConfiguration `tlv8:"1"`
	SelectedVideoConfiguration   VideoConfiguration     `tlv8:"2"`
	SelectedAudioConfiguration   AudioConfiguration     `tlv8:"3"`
}

type SupportedVideoRecordingConfiguration struct {
	CodecConfiguration []VideoConfiguration `tlv8:"1"`
}

type SupportedAudioRecordingConfiguration struct {
	CodecConfiguration []AudioConfiguration `tlv8:"1"`
}

type SupportedDataStreamTransportConfiguration struct {
	TransferTransportConfigurations []TransferTransportConfiguration `tlv8:"1"`
}

type TransferTransportConfiguration struct {
	TransportType byte `tlv8:"1"`
}

type RecordingConfiguration struct {
	PrebufferLength              uint16                        `tlv8:"1"`
	EventTriggerOptions          uint8                         `tlv8:"2"`
	MediaContainerConfigurations []MediaContainerConfiguration `tlv8:"3"`
}

type MediaContainerConfiguration struct {
	MediaContainerType       uint8                      `tlv8:"1"` // 0 - fragmented MP4
	MediaContainerParameters []MediaContainerParameters `tlv8:"2"`
}

type MediaContainerParameters struct {
	FragmentLength uint16 `tlv8:"1"`
}

type VideoConfiguration struct {
	Codec                uint8                  `tlv8:"1"` // 0 (H.264), 1 (H.265)
	VideoCodecParameters []VideoCodecParameters `tlv8:"2"`
	VideoAttributes      []VideoAttributes      `tlv8:"3"`
}

type VideoCodecParameters struct {
	ProfileID uint8 `tlv8:"1"` // 0 (baseline), 1 (main), 2 (high)
	Level     uint8 `tlv8:"2"` // 0 (3.1), 1 (3.2), 2 (4)
}

type VideoAttributes struct {
	ImageWidth  uint16 `tlv8:"1"`
	ImageHeight uint16 `tlv8:"2"`
	FrameRate   uint16 `tlv8:"3"`
}

type AudioConfiguration struct {
	Codec                uint8                  `tlv8:"1"` // 0 (AAC-LC), 1 (AAC-ELD)
	AudioCodecParameters []AudioCodecParameters `tlv8:"2"`
}

type AudioCodecParameters struct {
	Channels     uint8   `tlv8:"1"`
	BitrateModes []uint8 `tlv8:"2"` // 0 (variable), 1 (constant)
	SampleRates  []byte  `tlv8:"3"`
}
