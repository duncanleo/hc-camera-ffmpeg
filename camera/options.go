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
