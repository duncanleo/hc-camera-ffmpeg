package camera

import "github.com/brutella/hc/rtp"

func streamVideoProfile(cfg rtp.StreamConfiguration) string {
	switch cfg.Video.CodecParams.Profiles[0].Id {
	case rtp.VideoCodecProfileMain:
		return "main"
	default:
		return "high"
	}
}

func streamVideoCodecLevel(cfg rtp.StreamConfiguration) string {
	switch cfg.Video.CodecParams.Levels[0].Level {
	case rtp.VideoCodecLevel3_1:
		return "3.1"
	case rtp.VideoCodecLevel3_2:
		return "3.2"
	default:
		return "4.0"
	}
}

func streamVideoMTP(se rtp.SetupEndpoints) int {
	switch se.ControllerAddr.IPVersion {
	case rtp.IPAddrVersionv4:
		return 1378
	default:
		return 1228
	}
}

func streamAudioSampleRate(cfg rtp.StreamConfiguration) int {
	switch cfg.Audio.CodecParams.Samplerate {
	case rtp.AudioCodecSampleRate16Khz:
		return 16000
	case rtp.AudioCodecSampleRate24Khz:
		return 24000
	default:
		return 8000
	}
}

func streamAudioCodec(cfg rtp.StreamConfiguration) string {
	switch cfg.Audio.CodecType {
	case rtp.AudioCodecType_AAC_ELD:
		return "libfdk_aac"
	case rtp.AudioCodecType_Opus:
		return "libopus"
	default:
		return "mp3" // This should never work
	}
}
